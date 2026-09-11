'use client';

import { Suspense, useEffect } from 'react';
import { AppLoading } from '@/components/ui/AppLoading';
import { useRouter, useSearchParams } from 'next/navigation';

/**
 * Legacy URL: /appointments/new?carId=…
 * Scheduling now lives in a modal on /appointments.
 */
function RedirectInner() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const carId = searchParams.get('carId');
    const q = carId
      ? `?schedule=1&carId=${encodeURIComponent(carId)}`
      : '?schedule=1';
    router.replace(`/appointments${q}`);
  }, [router, searchParams]);

  return (
    <div className="loadingScreen" aria-busy="true">
      <AppLoading size="lg" aria-busy={false} label="A redirecionar" />
    </div>
  );
}

export default function LegacyNewAppointmentPage() {
  return (
    <Suspense
      fallback={
        <div className="loadingScreen" aria-busy="true">
          <AppLoading size="lg" aria-busy={false} label="A carregar" />
        </div>
      }
    >
      <RedirectInner />
    </Suspense>
  );
}
