'use client';

import React, { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import { BrandLogo } from '@/components/brand/BrandLogo';
import { BrandHeroBanner } from '@/components/brand/BrandHeroBanner';
import styles from './landing.module.css';
import { AppLoading } from '@/components/ui/AppLoading';

export default function LandingPage() {
  const { isAuthenticated, user, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && isAuthenticated && user) {
      router.push('/dashboard');
    }
  }, [isAuthenticated, user, isLoading, router]);

  const handleLogin = () => {
    router.push('/auth/login');
  };

  const handleRegister = () => {
    router.push('/auth/register');
  };

  const services = [
    {
      icon: '🔧',
      title: 'Manutenção geral',
      description: 'Revisões periódicas para manter o seu automóvel em segurança e bom estado.',
      price: 'Desde 89 €',
    },
    {
      icon: '🛞',
      title: 'Pneus e geometria',
      description: 'Rotação, alinhamento, equilíbrio e substituição de pneus.',
      price: 'Desde 45 €',
    },
    {
      icon: '🔋',
      title: 'Bateria',
      description: 'Teste, manutenção e substituição de baterias.',
      price: 'Desde 120 €',
    },
    {
      icon: '❄️',
      title: 'Climatização',
      description: 'Reparação, manutenção e recarga de gás do sistema A/C.',
      price: 'Desde 150 €',
    },
    {
      icon: '🚗',
      title: 'Diagnóstico de motor',
      description: 'Análise completa e localização de avarias.',
      price: 'Desde 95 €',
    },
    {
      icon: '🛡️',
      title: 'Travões',
      description: 'Pastilhas, discos e revisão integral do sistema de travagem.',
      price: 'Desde 180 €',
    },
  ];

  const features = [
    {
      icon: '👨‍🔧',
      title: 'Técnicos experientes',
      description: 'Mecânicos qualificados com experiência em oficina.',
    },
    {
      icon: '⚡',
      title: 'Serviço ágil',
      description: 'Reparações eficientes para voltar à estrada o mais depressa possível.',
    },
    {
      icon: '💰',
      title: 'Preços transparentes',
      description: 'Orçamentos claros, sem surpresas na fatura.',
    },
    {
      icon: '🛡️',
      title: 'Garantia de qualidade',
      description: 'Garantia de 90 dias nos nossos serviços.',
    },
  ];

  if (isLoading) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A carregar" />
      </div>
    );
  }

  if (isAuthenticated && user) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <div className="loadingScreenInner">
          <AppLoading size="lg" aria-busy={false} label="A redirecionar para o painel" />
          <p>A redirecionar para o painel…</p>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <div className={styles.headerContent}>
          <div className={styles.logoSection} onClick={() => router.push('/')}>
            <div className={styles.logo}>
              <BrandLogo alt="Logótipo GonsGarage" width={32} height={32} style={{ objectFit: 'contain' }} />
            </div>
            <div className={styles.logoText}>
              <h1>GonsGarage</h1>
              <p>Oficina mecânica profissional</p>
            </div>
          </div>

          <nav className={styles.navigation} aria-label="Secções">
            <a href="#services" className={styles.navLink}>
              Serviços
            </a>
            <a href="#about" className={styles.navLink}>
              Sobre
            </a>
            <a href="#contact" className={styles.navLink}>
              Contacto
            </a>
          </nav>

          <div className={styles.authButtons}>
            <button type="button" onClick={handleLogin} className={styles.loginButton}>
              Iniciar sessão
            </button>
            <button type="button" onClick={handleRegister} className={styles.registerButton}>
              Registar
            </button>
          </div>
        </div>
      </header>

      <section className={styles.hero}>
        <div className={styles.heroBackground}>
          <div className={styles.bannerImage}>
            <BrandHeroBanner alt="Oficina GonsGarage" priority />
          </div>
          <div className={styles.heroOverlay} />
        </div>

        <div className={styles.heroContent}>
          <div className={styles.heroText}>
            <h2>A sua oficina de confiança</h2>
            <p>
              Mecânicos experientes, serviço de qualidade e preços justos. Mantemos o seu automóvel em
              excelente estado, com o profissionalismo que espera.
            </p>
            <div className={styles.heroStats}>
              <div className={styles.stat}>
                <span className={styles.statNumber}>15+</span>
                <span className={styles.statLabel}>Anos de experiência</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statNumber}>5000+</span>
                <span className={styles.statLabel}>Clientes satisfeitos</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statNumber}>24/7</span>
                <span className={styles.statLabel}>Assistência urgente</span>
              </div>
            </div>
            <div className={styles.heroActions}>
              <button type="button" onClick={handleRegister} className={styles.ctaButton}>
                Agendar serviço
              </button>
              <button type="button" onClick={handleLogin} className={styles.secondaryButton}>
                Já sou cliente
              </button>
            </div>
          </div>
        </div>
      </section>

      <section id="services" className={styles.servicesSection}>
        <div className={styles.sectionContent}>
          <div className={styles.sectionHeader}>
            <h3>Os nossos serviços</h3>
            <p>Assistência completa a todas as marcas e modelos</p>
          </div>

          <div className={styles.servicesGrid}>
            {services.map((service, index) => (
              <div key={index} className={styles.serviceCard}>
                <div className={styles.serviceIcon}>{service.icon}</div>
                <h4>{service.title}</h4>
                <p>{service.description}</p>
                <div className={styles.servicePrice}>{service.price}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section id="about" className={styles.featuresSection}>
        <div className={styles.sectionContent}>
          <div className={styles.sectionHeader}>
            <h3>Porquê a GonsGarage?</h3>
            <p>Compromisso com um serviço automóvel de excelência</p>
          </div>

          <div className={styles.featuresGrid}>
            {features.map((feature, index) => (
              <div key={index} className={styles.featureCard}>
                <div className={styles.featureIcon}>{feature.icon}</div>
                <h4>{feature.title}</h4>
                <p>{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section id="contact" className={styles.contactSection}>
        <div className={styles.sectionContent}>
          <div className={styles.contactGrid}>
            <div className={styles.contactInfo}>
              <h3>Visite a nossa oficina</h3>
              <div className={styles.contactItem}>
                <div className={styles.contactIcon}>📍</div>
                <div>
                  <h4>Morada</h4>
                  <p>
                    Rua Exemplo, 123
                    <br />
                    1000-001 Lisboa, Portugal
                  </p>
                </div>
              </div>
              <div className={styles.contactItem}>
                <div className={styles.contactIcon}>📞</div>
                <div>
                  <h4>Telefone</h4>
                  <p>+351 21 123 4567</p>
                </div>
              </div>
              <div className={styles.contactItem}>
                <div className={styles.contactIcon}>🕒</div>
                <div>
                  <h4>Horário</h4>
                  <p>
                    Seg–Sex: 8h00–18h00
                    <br />
                    Sáb: 8h00–13h00
                    <br />
                    Dom: encerrado
                  </p>
                </div>
              </div>
            </div>

            <div className={styles.contactCta}>
              <h3>Pronto para começar?</h3>
              <p>Marque já a sua visita e conheça a diferença GonsGarage.</p>
              <button type="button" onClick={handleRegister} className={styles.ctaButton}>
                Marcar serviço
              </button>
            </div>
          </div>
        </div>
      </section>

      <footer className={styles.footer}>
        <div className={styles.footerContent}>
          <div className={styles.footerSection} onClick={() => router.push('/')}>
            <div className={styles.footerLogo}>
              <div className={styles.logo}>
                <BrandLogo alt="Logótipo GonsGarage" width={32} height={32} style={{ objectFit: 'contain' }} />
              </div>
              <div>
                <h4>GonsGarage</h4>
                <p>Oficina mecânica profissional</p>
              </div>
            </div>
          </div>

          <div className={styles.footerSection}>
            <h4>Serviços</h4>
            <ul>
              <li>Mudança de óleo</li>
              <li>Travões</li>
              <li>Pneus</li>
              <li>Motor</li>
            </ul>
          </div>

          <div className={styles.footerSection}>
            <h4>Empresa</h4>
            <ul>
              <li>Sobre nós</li>
              <li>Equipa</li>
              <li>Recrutamento</li>
              <li>Contacto</li>
            </ul>
          </div>

          <div className={styles.footerSection}>
            <h4>Cliente</h4>
            <ul>
              <li>
                <button type="button" onClick={handleLogin}>
                  Iniciar sessão
                </button>
              </li>
              <li>
                <button type="button" onClick={handleRegister}>
                  Registar
                </button>
              </li>
              <li>Histórico de serviços</li>
              <li>Ajuda</li>
            </ul>
          </div>
        </div>

        <div className={styles.footerBottom}>
          <p>&copy; {new Date().getFullYear()} GonsGarage. Todos os direitos reservados.</p>
        </div>
      </footer>
    </div>
  );
}
