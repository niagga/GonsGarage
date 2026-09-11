package main

import (
	"context"
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
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/sqlxdb"
	redisRepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/redis"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/appointment"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/auth"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/billing_document"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/car"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/employee"
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

	// Bases creadas antes de domain.Repair.technician_id: AutoMigrate puede haber fallado y el repo sqlx asume la columna.
	if err := ensureRepairsTechnicianIDColumn(db); err != nil {
		log.Fatalf("repairs.technician_id schema fix: %v", err)
	}
	if err := ensureRepairsServiceJobIDColumn(db); err != nil {
		log.Fatalf("repairs.service_job_id schema: %v", err)
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

	log.Printf("Use cases initialized")

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	adminUserHandler := handler.NewAdminUserHandler(authService)
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
		supplierHandler, receivedInvoiceHandler, billingDocumentHandler, invoiceHandler, partHandler,
		authMiddleware, sqlxDB)

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

// ensureRepairsTechnicianIDColumn alinea esquemas antiguos con el modelo actual (sqlx SELECT incluye technician_id).
func ensureRepairsTechnicianIDColumn(db *gorm.DB) error {
	const qAdd = `ALTER TABLE repairs ADD COLUMN IF NOT EXISTS technician_id uuid`
	const qFill = `UPDATE repairs SET technician_id = '00000000-0000-0000-0000-000000000000'::uuid WHERE technician_id IS NULL`
	const qDefault = `ALTER TABLE repairs ALTER COLUMN technician_id SET DEFAULT '00000000-0000-0000-0000-000000000000'::uuid`
	const qNotNull = `ALTER TABLE repairs ALTER COLUMN technician_id SET NOT NULL`
	for _, q := range []string{qAdd, qFill, qDefault, qNotNull} {
		if err := db.Exec(q).Error; err != nil {
			return fmt.Errorf("%s: %w", q, err)
		}
	}
	return nil
}

// ensureRepairsServiceJobIDColumn adds optional link from repairs to service_jobs (visits).
func ensureRepairsServiceJobIDColumn(db *gorm.DB) error {
	const q = `ALTER TABLE repairs ADD COLUMN IF NOT EXISTS service_job_id uuid`
	if err := db.Exec(q).Error; err != nil {
		return fmt.Errorf("%s: %w", q, err)
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
	authMiddleware *middleware.AuthMiddleware,
	sqlxDB *sqlx.DB,
) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "GonsGarage API is running",
		})
	})

	// Readiness: PostgreSQL via sqlx (shared pool with GORM)
	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := sqlxDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"db":     err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
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
		suppliers.Use(middleware.RequireWorkshopStaff())
		{
			suppliers.POST("", supplierHandler.CreateSupplier)
			suppliers.GET("", supplierHandler.ListSuppliers)
			suppliers.GET("/:id", supplierHandler.GetSupplier)
			suppliers.PUT("/:id", supplierHandler.UpdateSupplier)
			suppliers.DELETE("/:id", supplierHandler.DeleteSupplier)
		}

		receivedInvoices := protected.Group("/received-invoices")
		receivedInvoices.Use(middleware.RequireWorkshopStaff())
		{
			receivedInvoices.POST("", receivedInvoiceHandler.CreateReceivedInvoice)
			receivedInvoices.GET("", receivedInvoiceHandler.ListReceivedInvoices)
			receivedInvoices.GET("/:id", receivedInvoiceHandler.GetReceivedInvoice)
			receivedInvoices.PUT("/:id", receivedInvoiceHandler.UpdateReceivedInvoice)
			receivedInvoices.DELETE("/:id", receivedInvoiceHandler.DeleteReceivedInvoice)
		}

		billingDocs := protected.Group("/billing-documents")
		billingDocs.Use(middleware.RequireWorkshopStaff())
		{
			billingDocs.POST("", billingDocumentHandler.CreateBillingDocument)
			billingDocs.GET("", billingDocumentHandler.ListBillingDocuments)
			billingDocs.GET("/:id", billingDocumentHandler.GetBillingDocument)
			billingDocs.PUT("/:id", billingDocumentHandler.UpdateBillingDocument)
			billingDocs.DELETE("/:id", billingDocumentHandler.DeleteBillingDocument)
		}

		invoices := protected.Group("/invoices")
		{
			invoices.GET("/me", invoiceHandler.ListMyInvoices)
			staffInvoices := invoices.Group("")
			staffInvoices.Use(middleware.RequireWorkshopStaff())
			{
				staffInvoices.POST("", invoiceHandler.CreateIssuedInvoice)
				staffInvoices.GET("", invoiceHandler.ListIssuedInvoicesStaff)
				staffInvoices.DELETE("/:id", invoiceHandler.DeleteIssuedInvoice)
			}
			invoices.GET("/:id", invoiceHandler.GetIssuedInvoice)
			invoices.PATCH("/:id", invoiceHandler.PatchIssuedInvoice)
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
