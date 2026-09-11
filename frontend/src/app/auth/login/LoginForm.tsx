'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import { useAuth } from '@/stores';
import { AuthShell, AuthShellFooter } from '@/components/auth/AuthShell';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import styles from './login.module.css';

export default function LoginForm() {
  const [formData, setFormData] = useState({
    email: '',
    password: '',
  });
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');

  const { login } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const message = searchParams.get('message');
    if (!message) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setSuccessMessage(message);
    });
    return () => {
      cancelled = true;
    };
  }, [searchParams]);

  const validateForm = () => {
    const newErrors: { [key: string]: string } = {};

    if (!formData.email) {
      newErrors.email = 'O e-mail é obrigatório';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = 'E-mail inválido';
    }

    if (!formData.password) {
      newErrors.password = 'A palavra-passe é obrigatória';
    } else if (formData.password.length < 6) {
      newErrors.password = 'A palavra-passe deve ter pelo menos 6 caracteres';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsLoading(true);
    setErrors({});

    try {
      const result = await login(formData.email, formData.password);

      if (result.success) {
        router.replace('/dashboard');
      } else {
        setErrors({ general: result.error || 'Falha no início de sessão' });
      }
    } catch {
      setErrors({ general: 'Ocorreu um erro inesperado' });
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));

    if (errors[name]) {
      setErrors((prev) => ({
        ...prev,
        [name]: '',
      }));
    }
  };

  let banner: { variant: 'success' | 'error'; message: string } | null = null;
  if (errors.general) {
    banner = { variant: 'error', message: errors.general };
  } else if (successMessage) {
    banner = { variant: 'success', message: successMessage };
  }

  return (
    <AuthShell
      title="Iniciar sessão"
      subtitle="Aceda com a sua conta GonsGarage"
      banner={banner}
    >
      <form className={styles.form} onSubmit={handleSubmit}>
        <div className={styles.fieldStack}>
          <div className="grid gap-2">
            <Label htmlFor="email">E-mail</Label>
            <Input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              value={formData.email}
              onChange={handleChange}
              placeholder="O seu e-mail"
              aria-invalid={Boolean(errors.email)}
              aria-describedby={errors.email ? 'email-error' : undefined}
              className={cn(errors.email && 'border-destructive')}
            />
            {errors.email ? (
              <p id="email-error" className="text-sm text-destructive">
                {errors.email}
              </p>
            ) : null}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="password">Palavra-passe</Label>
            <div className="relative flex gap-1">
              <Input
                id="password"
                name="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                required
                value={formData.password}
                onChange={handleChange}
                placeholder="A sua palavra-passe"
                aria-invalid={Boolean(errors.password)}
                aria-describedby={errors.password ? 'password-error' : undefined}
                className={cn('pr-10', errors.password && 'border-destructive')}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0 top-0 h-9 w-9 shrink-0"
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? 'Ocultar palavra-passe' : 'Mostrar palavra-passe'}
              >
                {showPassword ? (
                  <span className="sr-only">Ocultar</span>
                ) : (
                  <span className="sr-only">Mostrar</span>
                )}
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden>
                  {showPassword ? (
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L7.05 7.05m2.828 2.828L12 12m2.122 2.122L17.95 17.95"
                    />
                  ) : (
                    <>
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                      />
                    </>
                  )}
                </svg>
              </Button>
            </div>
            {errors.password ? (
              <p id="password-error" className="text-sm text-destructive">
                {errors.password}
              </p>
            ) : null}
          </div>
        </div>

        <Button type="submit" size="lg" disabled={isLoading} className={styles.submitWide}>
          {isLoading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              A iniciar…
            </>
          ) : (
            'Iniciar sessão'
          )}
        </Button>

        <AuthShellFooter>
          <p className="text-center text-sm text-muted-foreground">
            Ainda não tem conta?{' '}
            <Button type="button" variant="link" className="h-auto p-0" onClick={() => router.push('/auth/register')}>
              Criar conta
            </Button>
          </p>
          <p className={styles.demoBlock}>
            Conta de demonstração: admin@gonsgarage.com / admin123
          </p>
        </AuthShellFooter>
      </form>
    </AuthShell>
  );
}
