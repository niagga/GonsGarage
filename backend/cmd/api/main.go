package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	postgresRepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/handler"
	fiscalmock "github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/mock"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	fiscalcrypto "github.com/gaston-garcia-cegid/gonsgarage/internal/platform/crypto"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/sqlxdb"
	redisRepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/redis"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/appointment"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/auth"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/billing_document"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/car"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/employee"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/invoice"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/part"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/received_invoice"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/repair"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/servicejob"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/supplier"

	_ "github.com/gaston-garcia-cegid/gonsgarage/docs" // swagger (swag)
)

// @title           GonsGarage API
// @version         1.0
// @description     API de gestión de taller: autenticación JWT, coches, citas, empleados, proveedores, facturas recibidas/emitidas y documentos de facturación.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     JWT: cabecera Authorization con valor Bearer seguido del token (rutas bajo /api/v1 salvo /health).

// redactDatabaseURL oculta la contraseña del DSN para logs (nunca loguear credenciales en prod).
func redactDatabaseURL(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Sprintf("(dsn no parseable: %v)", err)
	}
	return u.Redacted()
}

func main() {
	log.Printf("/*************** Start Main ***************/")

	// Database connection with timeout
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admindb:gonsgarage123@localhost:5432/gonsgarage?sslmode=disable"
	}
	log.Printf("Connecting to database: %s", redactDatabaseURL(dsn))

	// Configure GORM with better logging and timeout
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true,
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test database connection with timeout
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Printf("Database connection established")

	sqlxDB := sqlxdb.WrapPostgres(sqlDB)
	if sqlxDB == nil {
		log.Fatal("sqlx wrap failed: nil *sql.DB")
	}
	// Fiscal tables are explicit migrations, never AutoMigrate. Legacy startup is
	// unchanged while the additive capability is disabled.
	fiscalFeatureEarly := fiscalsvc.ParseRuntimeConfig(os.Getenv).FeatureEnabled
	if fiscalFeatureEarly {
		schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer schemaCancel()
		if err := postgresRepo.VerifyFiscalSchema(schemaCtx, sqlDB); err != nil {
			log.Fatalf("fiscal capability requires migration 011: %v", err)
		}
	}

	// Check if we need to reset database (for development)
	resetDB := os.Getenv("RESET_DATABASE")
	if resetDB == "true" {
		log.Printf("RESET_DATABASE=true, dropping existing tables...")
		if err := dropAllTables(db); err != nil {
			log.Printf("Warning: Failed to drop tables: %v", err)
		}
	}

	// Auto-migrate tables in correct order (dependencies first)
	log.Printf("Starting database migration...")

	// Migrate tables one by one in dependency order
	models := []interface{}{
		&domain.User{},
		&domain.Employee{},
		&domain.Car{},
		&domain.Repair{},
		&domain.ServiceJob{},
		&domain.ServiceJobReception{},
		&domain.ServiceJobHandover{},
		&domain.Appointment{},
		&domain.PartItem{},
		&domain.Supplier{},
		&domain.ReceivedInvoice{},
		&domain.BillingDocument{},
		&domain.Invoice{},
	}

	for _, model := range models {
		log.Printf("Migrating %T...", model)
		if err := db.AutoMigrate(model); err != nil {
			log.Printf("Migration error for %T: %v", model, err)
			// Continue with other migrations instead of failing completely
			continue
		}
		log.Printf("Successfully migrated %T", model)
	}

	// Legacy repairs tables (start_date/end_date, employee_id) drift from sqlx SELECT;
	// AutoMigrate often fails silently (continue above). Ensure* adds/backfills required columns.
	if err := postgresRepo.EnsureRepairsSchema(db); err != nil {
		log.Fatalf("repairs schema fix: %v", err)
	}

	// Create indexes manually if they don't exist
	if err := createIndexes(db); err != nil {
		log.Printf("Warning: Failed to create some indexes: %v", err)
	}

	log.Printf("Database migration completed successfully")

	// Redis connection with timeout
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test Redis connection with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cacheRepo ports.CacheRepository
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Printf("Continuing without Redis cache...")
		cacheRepo = &NullCacheRepository{}
	} else {
		log.Printf("Redis connection established")
		cacheRepo = redisRepo.NewRedisCacheRepository(rdb)
	}

	// Initialize repositories
	userRepo := postgresRepo.NewPostgresUserRepository(db)
	employeeRepo := postgresRepo.NewPostgresEmployeeRepository(db)
	carRepo := postgresRepo.NewPostgresCarRepository(db)
	appointmentRepo := postgresRepo.NewPostgresAppointmentRepository(db)
	repairRepo := postgresRepo.NewPostgresRepairRepository(db)
	serviceJobRepo := postgresRepo.NewPostgresServiceJobRepository(db)
	supplierRepo := postgresRepo.NewPostgresSupplierRepository(db)
	receivedInvoiceRepo := postgresRepo.NewPostgresReceivedInvoiceRepository(db)
	billingDocRepo := postgresRepo.NewPostgresBillingDocumentRepository(db)
	invoiceRepo := postgresRepo.NewPostgresInvoiceRepository(db)
	partItemRepo := postgresRepo.NewPostgresPartItemRepository(db)
	log.Printf("Repositories initialized")

	// Initialize use cases
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-super-secret-jwt-key"
		log.Printf("Warning: Using default JWT secret. Set JWT_SECRET environment variable in production.")
	}

	authService := auth.NewAuthService(userRepo, jwtSecret, 24)
	employeeService := employee.NewEmployeeService(employeeRepo, cacheRepo)
	carService := car.NewCarService(carRepo, userRepo, cacheRepo)
	appointmentService := appointment.NewAppointmentService(appointmentRepo, userRepo, carRepo)
	repairService := repair.NewRepairService(repairRepo, carRepo, userRepo)
	serviceJobService := servicejob.NewService(serviceJobRepo, carRepo, userRepo, repairRepo)
	supplierService := supplier.NewSupplierService(supplierRepo, userRepo)
	receivedInvoiceService := received_invoice.NewReceivedInvoiceService(receivedInvoiceRepo, userRepo)
	billingDocumentService := billing_document.NewBillingDocumentService(billingDocRepo, userRepo)
	invoiceService := invoice.NewInvoiceService(invoiceRepo, userRepo)
	partService := part.NewPartService(partItemRepo, userRepo)

	var fiscalIntegrationHandler *handler.FiscalIntegrationHandler
	var fiscalDeps func() fiscalsvc.FiscalDependencyStatus
	fiscalRuntime := fiscalsvc.ParseRuntimeConfig(os.Getenv)
	fiscalRuntime.JWTSecretIsDefault = jwtSecret == "" || jwtSecret == "your-super-secret-jwt-key" || strings.HasPrefix(jwtSecret, "CHANGE_ME")
	if fiscalRuntime.FeatureEnabled {
		schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 5*time.Second)
		fiscalRuntime.SchemaPresent = postgresRepo.VerifyFiscalSchema(schemaCtx, sqlDB) == nil
		schemaCancel()
		if err := validateAPIFiscalComposition(fiscalRuntime); err != nil {
			log.Fatalf("fiscal production composition rejected: %v", err)
		}
		providerKey, providerErr := fiscalsvc.ResolveProviderKey(fiscalRuntime)
		if providerErr != nil {
			log.Fatalf("fiscal provider configuration rejected: %v", providerErr)
		}
		fiscalRuntime.ProviderKey = providerKey

		fiscalConnectionRepo := postgresRepo.NewPostgresFiscalConnectionRepository(db)
		credentialKeyVersion := fiscalRuntime.CredentialKeyVersion
		var credentialKey []byte
		if encoded := strings.TrimSpace(os.Getenv("FISCAL_CREDENTIAL_KEY_B64")); encoded != "" {
			decoded, decodeErr := base64.StdEncoding.DecodeString(encoded)
			if decodeErr != nil {
				log.Printf("Warning: fiscal integration disabled: invalid FISCAL_CREDENTIAL_KEY_B64: %v", decodeErr)
			} else {
				credentialKey = decoded
			}
		} else if rawKey := strings.TrimSpace(os.Getenv("FISCAL_CREDENTIAL_KEY")); rawKey != "" {
			credentialKey = []byte(rawKey)
		}
		providerHealthy := false
		if len(credentialKey) == 0 {
			log.Printf("Warning: fiscal integration disabled: missing FISCAL_CREDENTIAL_KEY or FISCAL_CREDENTIAL_KEY_B64")
		} else if cipher, cipherErr := fiscalcrypto.NewFiscalCredentialCipher(credentialKeyVersion, credentialKey); cipherErr != nil {
			log.Printf("Warning: fiscal integration disabled: %v", cipherErr)
		} else if keyringErr := fiscalcrypto.ValidateProductionKeyring(fiscalRuntime.AppEnv, credentialKeyVersion, credentialKey); keyringErr != nil {
			log.Printf("Warning: fiscal integration disabled: %v", keyringErr)
		} else if providerKey == "mock" {
			if guardErr := fiscalmock.AssertMockAllowed(fiscalRuntime.AppEnv); guardErr != nil {
				log.Printf("Warning: fiscal mock provider not wired: %v", guardErr)
			} else if classErr := fiscalmock.RejectLegalMockClassification(fiscalRuntime.AppEnv, "mock"); classErr != nil {
				log.Printf("Warning: fiscal mock provider not wired: %v", classErr)
			} else if mockProvider, providerErr := fiscalmock.NewProvider(fiscalmock.Options{
				AppEnv:   fiscalRuntime.AppEnv,
				Registry: fiscalmock.NewScenarioRegistry(nil),
				Store:    fiscalmock.NewPostgresOperationStore(sqlDB),
			}); providerErr != nil {
				log.Printf("Warning: fiscal integration disabled: %v", providerErr)
			} else {
				fiscalService := fiscalsvc.NewConnectionService(fiscalConnectionRepo, userRepo, mockProvider, cipher, credentialKeyVersion)
				fiscalIntegrationHandler = handler.NewFiscalIntegrationHandler(fiscalService)
				providerHealthy = true
				log.Printf("Fiscal integration foundation wired with deterministic mock provider (env=%s, classification=mock)", fiscalRuntime.AppEnv)
			}
		} else {
			log.Printf("Fiscal provider %q selected; connection routes require Cloudware composition (mutations default off)", providerKey)
		}
		fiscalDeps = func() fiscalsvc.FiscalDependencyStatus {
			return fiscalsvc.BuildFiscalDependencyStatus(fiscalsvc.DependencyStatusInput{
				FeatureEnabled:      fiscalRuntime.FeatureEnabled,
				FinalizationEnabled: fiscalRuntime.FinalizationEnabled,
				WorkerEnabled:       fiscalRuntime.WorkerEnabled,
				ProviderKey:         fiscalRuntime.ProviderKey,
				ProviderHealthy:     providerHealthy,
				SchemaPresent:       fiscalRuntime.SchemaPresent,
			})
		}
	} else {
		fiscalDeps = func() fiscalsvc.FiscalDependencyStatus {
			return fiscalsvc.BuildFiscalDependencyStatus(fiscalsvc.DependencyStatusInput{
				FeatureEnabled: false, SchemaPresent: false, ProviderHealthy: true,
			})
		}
	}

	log.Printf("Use cases initialized")

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	adminUserHandler := handler.NewAdminUserHandler(authService, userRepo)
	employeeHandler := handler.NewEmployeeHandler(employeeService)
	carHandler := handler.NewCarHandler(carService)

	// Initialize appointment handler
	appointmentHandler := handler.NewAppointmentHandler(appointmentService)
	repairHandler := handler.NewRepairHandler(repairService)
	serviceJobHandler := handler.NewServiceJobHandler(serviceJobService)
	supplierHandler := handler.NewSupplierHandler(supplierService)
	receivedInvoiceHandler := handler.NewReceivedInvoiceHandler(receivedInvoiceService)
	billingDocumentHandler := handler.NewBillingDocumentHandler(billingDocumentService)
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	partHandler := handler.NewPartHandler(partService)

	var fiscalHandler *handler.FiscalHandler
	if fiscalRuntime.FeatureEnabled {
		fiscalRepo := postgresRepo.NewFiscalRepository(sqlDB)
		invoiceService = invoiceService.WithFiscalProtection(fiscalRepo)
		invoiceHandler = handler.NewInvoiceHandler(invoiceService)
		draftSvc := fiscalsvc.NewDraftService(fiscalRepo, invoiceRepo, userRepo)
		finalSvc := fiscalsvc.NewFinalizationService(fiscalRepo, invoiceRepo, userRepo).WithEnabled(fiscalRuntime.FinalizationEnabled)
		artifactRepo := postgresRepo.NewFiscalArtifactRepository(sqlDB)
		var artifactSvc *fiscalsvc.ArtifactService
		if art, artErr := fiscalsvc.ComposeArtifactService(fiscalRuntime.AppEnv, sqlDB, os.Getenv, fiscalRuntime.FinalizationEnabled); artErr != nil {
			log.Printf("Warning: fiscal artifact service not wired: %v", artErr)
		} else if art != nil {
			artifactSvc = art.Service
			log.Printf("Fiscal artifact service wired backend=%s env=%s", art.Backend, art.Environment)
		}
		docSvc := fiscalsvc.NewDocumentService(draftSvc, finalSvc, artifactSvc, fiscalRepo, invoiceRepo, userRepo, artifactRepo)
		fiscalHandler = handler.NewFiscalHandler(docSvc)
		log.Printf("Fiscal document HTTP APIs wired (finalization_enabled=%v)", fiscalRuntime.FinalizationEnabled)
	}

	log.Printf("Handlers initialized")

	// Setup router
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// CORS middleware
	router.Use(corsMiddleware())

	// Setup routes
	setupRoutes(router, authHandler, adminUserHandler, employeeHandler, carHandler, appointmentHandler, repairHandler, serviceJobHandler,
		supplierHandler, receivedInvoiceHandler, billingDocumentHandler, invoiceHandler, partHandler, fiscalIntegrationHandler, fiscalHandler,
		authMiddleware, sqlxDB, fiscalDeps)

	log.Printf("Routes set up")

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// Drop all tables for clean migration
func dropAllTables(db *gorm.DB) error {
	tables := []string{
		"appointments",
		"repairs",
		"received_invoices",
		"billing_documents",
		"invoices",
		"part_items",
		"suppliers",
		"cars",
		"employees",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE").Error; err != nil {
			log.Printf("Failed to drop table %s: %v", table, err)
		} else {
			log.Printf("Dropped table: %s", table)
		}
	}
	return nil
}

// Create indexes manually
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)",
		"CREATE INDEX IF NOT EXISTS idx_employees_user_id ON employees(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_employees_deleted_at ON employees(deleted_at)",
		"CREATE INDEX IF NOT EXISTS idx_cars_owner_id ON cars(owner_id)",
		"CREATE INDEX IF NOT EXISTS idx_cars_license_plate ON cars(license_plate)",
		"CREATE INDEX IF NOT EXISTS idx_cars_deleted_at ON cars(deleted_at)",
		"CREATE INDEX IF NOT EXISTS idx_repairs_car_id ON repairs(car_id)",
		"CREATE INDEX IF NOT EXISTS idx_repairs_technician_id ON repairs(technician_id)",
		"CREATE INDEX IF NOT EXISTS idx_repairs_deleted_at ON repairs(deleted_at)",
		"CREATE INDEX IF NOT EXISTS idx_repairs_service_job_id ON repairs(service_job_id)",
		"CREATE INDEX IF NOT EXISTS idx_appointments_customer_id ON appointments(customer_id)",
		"CREATE INDEX IF NOT EXISTS idx_appointments_car_id ON appointments(car_id)",
		"CREATE INDEX IF NOT EXISTS idx_appointments_deleted_at ON appointments(deleted_at)",
		"CREATE INDEX IF NOT EXISTS idx_service_jobs_car_id ON service_jobs(car_id)",
		"CREATE INDEX IF NOT EXISTS idx_service_jobs_opened_by_user_id ON service_jobs(opened_by_user_id)",
		"CREATE INDEX IF NOT EXISTS idx_service_jobs_deleted_at ON service_jobs(deleted_at)",
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Printf("Failed to create index: %s, error: %v", idx, err)
		}
	}
	return nil
}

// corsExtraOrigins parses CORS_ORIGINS (comma-separated) for GIN_MODE=release (e.g. LAN deploy).
func corsExtraOrigins() []string {
	v := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	if v == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// CORS middleware function
func corsMiddleware() gin.HandlerFunc {
	extras := corsExtraOrigins()
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		release := os.Getenv("GIN_MODE") == "release"

		allow := false
		if !release {
			allow = origin != ""
		} else {
			if origin == "http://localhost:3000" || origin == "http://localhost:3001" {
				allow = true
			}
			if !allow {
				for _, o := range extras {
					if o == origin {
						allow = true
						break
					}
				}
			}
		}
		if allow {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Setup all routes
func setupRoutes(
	router *gin.Engine,
	authHandler *handler.AuthHandler,
	adminUserHandler *handler.AdminUserHandler,
	employeeHandler *handler.EmployeeHandler,
	carHandler *handler.CarHandler,
	appointmentHandler *handler.AppointmentHandler,
	repairHandler *handler.RepairHandler,
	serviceJobHandler *handler.ServiceJobHandler,
	supplierHandler *handler.SupplierHandler,
	receivedInvoiceHandler *handler.ReceivedInvoiceHandler,
	billingDocumentHandler *handler.BillingDocumentHandler,
	invoiceHandler *handler.InvoiceHandler,
	partHandler *handler.PartHandler,
	fiscalIntegrationHandler *handler.FiscalIntegrationHandler,
	fiscalHandler *handler.FiscalHandler,
	authMiddleware *middleware.AuthMiddleware,
	sqlxDB *sqlx.DB,
	fiscalDeps func() fiscalsvc.FiscalDependencyStatus,
) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "GonsGarage API is running",
		})
	})

	// Readiness: PostgreSQL via sqlx (shared pool with GORM). Provider health is isolated.
	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		dbOK := sqlxDB.PingContext(ctx) == nil
		providerHealthy := true
		if fiscalDeps != nil {
			providerHealthy = fiscalDeps().ProviderHealthy
		}
		if !fiscalsvc.CoreAPIReady(dbOK, providerHealthy) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Fiscal dependency status is separate from /ready so provider outages do not take the API out of rotation.
	router.GET("/fiscal/dependency-status", func(c *gin.Context) {
		if fiscalDeps == nil {
			c.JSON(http.StatusOK, fiscalsvc.FiscalDependencyStatus{ProviderHealthy: true})
			return
		}
		c.JSON(http.StatusOK, fiscalDeps())
	})

	// API v1 routes
	api := router.Group("/api/v1")

	// Public auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.GinBearerJWT(authMiddleware))
	{
		protected.GET("/auth/me", authHandler.Me)

		adminUsers := protected.Group("/admin")
		adminUsers.Use(middleware.RequireStaffManagers())
		{
			adminUsers.POST("/users", adminUserHandler.ProvisionUser)
			adminUsers.GET("/users/clients", adminUserHandler.ListClients)
		}

		// Employee routes (admin / manager only)
		employees := protected.Group("/employees")
		employees.Use(middleware.RequireStaffManagers())
		{
			employees.POST("", employeeHandler.CreateEmployee)
			employees.GET("", employeeHandler.ListEmployees)
			employees.GET("/:id", employeeHandler.GetEmployee)
			employees.PUT("/:id", employeeHandler.UpdateEmployee)
			employees.DELETE("/:id", employeeHandler.DeleteEmployee)
		}

		parts := protected.Group("/parts")
		parts.Use(middleware.RequireStaffManagers())
		{
			parts.POST("", partHandler.CreatePartItem)
			parts.GET("", partHandler.ListParts)
			parts.GET("/:id", partHandler.GetPartItem)
			parts.PATCH("/:id", partHandler.UpdatePartItem)
			parts.DELETE("/:id", partHandler.DeletePartItem)
		}

		// Car routes would go here
		cars := protected.Group("/cars")
		{
			cars.POST("", carHandler.CreateCar)
			cars.GET("", carHandler.ListCars)
			cars.GET("/:id", carHandler.GetCar)
			cars.PUT("/:id", carHandler.UpdateCar)
			cars.DELETE("/:id", carHandler.DeleteCar)
		}

		// Appointment routes would go here
		appointments := protected.Group("/appointments")
		{
			appointments.POST("", appointmentHandler.CreateAppointment)
			appointments.GET("", appointmentHandler.ListAppointments)
			appointments.GET("/:id", appointmentHandler.GetAppointment)
			appointments.PUT("/:id", appointmentHandler.UpdateAppointment)
			appointments.DELETE("/:id", appointmentHandler.DeleteAppointment)
		}

		repairs := protected.Group("/repairs")
		{
			repairs.GET("/car/:carId", repairHandler.ListRepairsByCar)
			repairs.POST("", repairHandler.GinCreateRepair)
			repairs.GET("/:id", repairHandler.GinGetRepair)
			repairs.PATCH("/:id", repairHandler.GinPatchRepair)
			repairs.DELETE("/:id", repairHandler.GinDeleteRepair)
		}

		svcJobs := protected.Group("/service-jobs")
		svcJobs.Use(middleware.RequireWorkshopStaff())
		{
			svcJobs.POST("", serviceJobHandler.CreateServiceJob)
			svcJobs.GET("", serviceJobHandler.ListServiceJobsByOpenedOn)
			svcJobs.GET("/car/:carId", serviceJobHandler.ListServiceJobsByCar)
			svcJobs.GET("/:id/obd", serviceJobHandler.StubOBD)
			svcJobs.GET("/:id", serviceJobHandler.GetServiceJob)
			svcJobs.PUT("/:id/reception", serviceJobHandler.PutReception)
			svcJobs.PUT("/:id/handover", serviceJobHandler.PutHandover)
		}

		suppliers := protected.Group("/suppliers")
		suppliers.Use(middleware.RequireAccountingAccess())
		{
			suppliers.POST("", supplierHandler.CreateSupplier)
			suppliers.GET("", supplierHandler.ListSuppliers)
			suppliers.GET("/:id", supplierHandler.GetSupplier)
			suppliers.PUT("/:id", supplierHandler.UpdateSupplier)
			suppliers.DELETE("/:id", supplierHandler.DeleteSupplier)
		}

		receivedInvoices := protected.Group("/received-invoices")
		receivedInvoices.Use(middleware.RequireAccountingAccess())
		{
			receivedInvoices.POST("", receivedInvoiceHandler.CreateReceivedInvoice)
			receivedInvoices.GET("", receivedInvoiceHandler.ListReceivedInvoices)
			receivedInvoices.GET("/:id", receivedInvoiceHandler.GetReceivedInvoice)
			receivedInvoices.PUT("/:id", receivedInvoiceHandler.UpdateReceivedInvoice)
			receivedInvoices.DELETE("/:id", receivedInvoiceHandler.DeleteReceivedInvoice)
		}

		billingDocs := protected.Group("/billing-documents")
		billingDocs.Use(middleware.RequireAccountingAccess())
		{
			billingDocs.POST("", billingDocumentHandler.CreateBillingDocument)
			billingDocs.GET("", billingDocumentHandler.ListBillingDocuments)
			billingDocs.GET("/:id", billingDocumentHandler.GetBillingDocument)
			billingDocs.PUT("/:id", billingDocumentHandler.UpdateBillingDocument)
			billingDocs.DELETE("/:id", billingDocumentHandler.DeleteBillingDocument)
		}

		handler.RegisterInvoiceAndFiscalRoutes(protected, invoiceHandler, fiscalHandler)

		if fiscalIntegrationHandler != nil {
			fiscal := protected.Group("/fiscal")
			fiscal.Use(middleware.RequireStaffManagers())
			{
				connections := fiscal.Group("/connections/:scopeKey/:providerKey")
				{
					connections.GET("", fiscalIntegrationHandler.Status)
					connections.PUT("", fiscalIntegrationHandler.SetupConnection)
					connections.POST("/verify", fiscalIntegrationHandler.VerifyConnection)
					connections.DELETE("", fiscalIntegrationHandler.RevokeConnection)
				}
			}
			cloudwareIntegrations := protected.Group("/fiscal-integrations/cloudware")
			cloudwareIntegrations.Use(middleware.RequireStaffManagers())
			{
				cloudwareIntegrations.GET("/readiness", fiscalIntegrationHandler.CloudwareReadiness)
				cloudwareIntegrations.GET("/connection", fiscalIntegrationHandler.CloudwareConnectionStatus)
				cloudwareIntegrations.POST("/connect", fiscalIntegrationHandler.CloudwareConnect)
				cloudwareIntegrations.POST("/verify", fiscalIntegrationHandler.CloudwareVerify)
				cloudwareIntegrations.POST("/disconnect", fiscalIntegrationHandler.CloudwareDisconnect)
			}
			// OAuth callback is state-protected; raw state binding replaces session cookie auth.
			api.GET("/fiscal-integrations/cloudware/oauth/callback", fiscalIntegrationHandler.CloudwareOAuthCallback)
		}
	}
}

// Helper function to get user ID from Gin context
func getUserIDFromGinContext(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	userIDString, ok := userIDStr.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("user ID is not a string")
	}

	return uuid.Parse(userIDString)
}

// ResponseWriter wrapper to capture status codes
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// NullCacheRepository é uma implementação dummy quando Redis não está disponível
type NullCacheRepository struct{}

func (n *NullCacheRepository) Get(ctx context.Context, key string, dest interface{}) error {
	return nil // Não faz nada
}

func (n *NullCacheRepository) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	return nil // Não faz nada
}

func (n *NullCacheRepository) Delete(ctx context.Context, key string) error {
	return nil // Não faz nada
}
