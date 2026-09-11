// src/app/client/page.tsx
'use client';

import React, { useState, useCallback } from 'react';
import { useAuthGuard } from '@hooks/useAuthGuard';
import DashboardLayout from '@/components/layouts/DashboardLayout';
import ClientCars from './components/ClientCars';
import { useClientData } from './hooks/useClientData';
import { Car } from '@/types/car';
import ClientDashboard from './components/ClientDashboard';
import ClientAppointments from './components/ClientAppointments';
import { AppLoading } from '@/components/ui/AppLoading';

type ActiveTab = 'dashboard' | 'cars' | 'appointments';

export default function ClientPage() {
  const { isLoading: authLoading } = useAuthGuard('client');
  const { cars, repairs, appointments, loading, error } = useClientData();
  const [activeTab, setActiveTab] = useState<ActiveTab>('dashboard');

  const [userCars, setUserCars] = useState<Car[]>([]);

  // Handle when a car is added - following Agent.md callback patterns
  const handleAddCar = useCallback((newCar: Car) => {
  console.log('New car added:', newCar);
  setUserCars(prevCars => [...prevCars, newCar]);
  alert(`${newCar.year} ${newCar.make} ${newCar.model} foi adicionado com sucesso.`);
}, []);

  // Handle when cars list is updated - following Agent.md state management
  const handleUpdateCar = useCallback((updatedCars: Car[]) => {
    setUserCars(updatedCars);

    // Optional: Sync with parent state or context
    // updateUserCarsContext(updatedCars);
  }, []);

  if (authLoading || loading) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A carregar área do cliente" />
      </div>
    );
  }

  const navigationItems = [
    { key: 'dashboard', label: 'Painel', href: '#' },
    { key: 'cars', label: 'Os meus automóveis', href: '#' },
    { key: 'appointments', label: 'Marcações', href: '#' },
  ];

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return (
          <ClientDashboard 
            error={error}
            cars={cars}  // ✅ Direct use, no mapping needed
            recentRepairs={repairs}
            upcomingAppointments={appointments
              .filter(a => new Date(a.date) > new Date())
            }
            onNavigate={(tab: string) => setActiveTab(tab as ActiveTab)}
          />
        );
      case 'cars':
        return <ClientCars 
        onAddCar={handleAddCar}
        onUpdateCar={handleUpdateCar}
        showAddButton={true}
        maxCars={5} />;
      case 'appointments':
        return <ClientAppointments />;
    }
  };

  const tabTitles: Record<ActiveTab, string> = {
    dashboard: 'Painel',
    cars: 'Automóveis',
    appointments: 'Marcações',
  };

  return (
    <DashboardLayout
      title={tabTitles[activeTab]}
      subtitle="Área do cliente"
      activeTab={activeTab}
      navigationItems={navigationItems}
      onNavClick={(tab: string) => setActiveTab(tab as ActiveTab)}
    >
      {renderContent()}
    </DashboardLayout>
  );
}