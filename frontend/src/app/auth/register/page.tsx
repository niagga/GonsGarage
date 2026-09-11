'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import { UserRole } from '@/types';
import { useAuth } from '@/stores';
import { AuthShell, AuthShellFooter } from '@/components/auth/AuthShell';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import styles from './register.module.css';

interface FormData {
  firstName: string;
  lastName: string;
  email: string;
  password: string;
  confirmPassword: string;
  role: string;
}

interface FormErrors {
  [key: string]: string;
}

export default function RegisterPage() {
  const [formData, setFormData] = useState<FormData>({
    firstName: '',
    lastName: '',
    email: '',
    password: '',
    confirmPassword: '',
    role: 'client',
  });

  const [errors, setErrors] = useState<FormErrors>({});
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const { register, isAuthenticated } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isAuthenticated) {
      router.push('/employees');
    }
  }, [isAuthenticated, router]);

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    if (!formData.firstName.trim()) {
      newErrors.firstName = 'O nome é obrigatório';
    } else if (formData.firstName.trim().length < 2) {
      newErrors.firstName = 'O nome deve ter pelo menos 2 caracteres';
    }

    if (!formData.lastName.trim()) {
      newErrors.lastName = 'O apelido é obrigatório';
    } else if (formData.lastName.trim().length < 2) {
      newErrors.lastName = 'O apelido deve ter pelo menos 2 caracteres';
    }

    if (!formData.email.trim()) {
      newErrors.email = 'O e-mail é obrigatório';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = 'Indique um e-mail válido';
    }

    if (!formData.password) {
      newErrors.password = 'A palavra-passe é obrigatória';
    } else if (formData.password.length < 6) {
      newErrors.password = 'A palavra-passe deve ter pelo menos 6 caracteres';
    }

    if (!formData.confirmPassword) {
      newErrors.confirmPassword = 'Confirme a palavra-passe';
    } else if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = 'As palavras-passe não coincidem';
    }

    if (!formData.role) {
      newErrors.role = 'Selecione um perfil';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsLoading(true);
    setErrors({});

    try {
      const result = await register({
        email: formData.email.trim(),
        password: formData.password,
        firstName: formData.firstName.trim(),
        lastName: formData.lastName.trim(),
        role: formData.role as UserRole,
      });

      if (result.success) {
        router.push('/auth/login?message=Registo concluído. Inicie sessão com as suas credenciais.');
      } else {
        setErrors({ general: result.error || 'O registo falhou. Tente novamente.' });
      }
    } catch (error) {
      let errorMessage = 'Ocorreu um erro inesperado. Tente novamente.';

      if (error instanceof Error) {
        errorMessage = error.message.trim() || errorMessage;
      } else if (typeof error === 'string') {
        errorMessage = error;
      }

      setErrors({ general: errorMessage });
    } finally {
      setIsLoading(false);
    }
  };

  const togglePasswordVisibility = (field: 'password' | 'confirmPassword') => {
    if (field === 'password') {
      setShowPassword(!showPassword);
    } else {
      setShowConfirmPassword(!showConfirmPassword);
    }
  };

  const banner =
    errors.general && errors.general.length > 0
      ? ({ variant: 'error' as const, message: errors.general })
      : null;

  const passwordField = (
    id: 'password' | 'confirmPassword',
    label: string,
    placeholder: string,
    autoComplete: string,
    show: boolean,
    onToggle: () => void,
    value: string,
    error?: string,
  ) => (
    <div className="grid gap-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="relative flex">
        <Input
          id={id}
          name={id}
          type={show ? 'text' : 'password'}
          value={value}
          onChange={handleChange}
          placeholder={placeholder}
          autoComplete={autoComplete}
          required
          aria-invalid={Boolean(error)}
          aria-describedby={error ? `${id}-error` : undefined}
          className={cn('pr-10', error && 'border-destructive')}
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="absolute right-0 top-0 h-9 w-9"
          onClick={onToggle}
          aria-label={show ? 'Ocultar palavra-passe' : 'Mostrar palavra-passe'}
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden>
            {show ? (
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
      {error ? (
        <p id={`${id}-error`} className="text-sm text-destructive">
          {error}
        </p>
      ) : null}
    </div>
  );

  return (
    <AuthShell title="Criar conta" subtitle="Junte-se à equipa GonsGarage" banner={banner}>
      <form className={styles.form} onSubmit={handleSubmit}>
        <div className={styles.fieldsContainer}>
          <div className={styles.nameFieldsRow}>
            <div className="grid gap-2">
              <Label htmlFor="firstName">Nome</Label>
              <Input
                id="firstName"
                name="firstName"
                type="text"
                value={formData.firstName}
                onChange={handleChange}
                placeholder="O seu nome"
                autoComplete="given-name"
                required
                aria-invalid={Boolean(errors.firstName)}
                className={cn(errors.firstName && 'border-destructive')}
              />
              {errors.firstName ? <p className="text-sm text-destructive">{errors.firstName}</p> : null}
            </div>

            <div className="grid gap-2">
              <Label htmlFor="lastName">Apelido</Label>
              <Input
                id="lastName"
                name="lastName"
                type="text"
                value={formData.lastName}
                onChange={handleChange}
                placeholder="O seu apelido"
                autoComplete="family-name"
                required
                aria-invalid={Boolean(errors.lastName)}
                className={cn(errors.lastName && 'border-destructive')}
              />
              {errors.lastName ? <p className="text-sm text-destructive">{errors.lastName}</p> : null}
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="email">E-mail</Label>
            <Input
              id="email"
              name="email"
              type="email"
              value={formData.email}
              onChange={handleChange}
              placeholder="O seu e-mail"
              autoComplete="email"
              required
              aria-invalid={Boolean(errors.email)}
              className={cn(errors.email && 'border-destructive')}
            />
            {errors.email ? <p className="text-sm text-destructive">{errors.email}</p> : null}
          </div>

          {passwordField(
            'password',
            'Palavra-passe',
            'Crie uma palavra-passe',
            'new-password',
            showPassword,
            () => togglePasswordVisibility('password'),
            formData.password,
            errors.password,
          )}

          {passwordField(
            'confirmPassword',
            'Confirmar palavra-passe',
            'Confirme a palavra-passe',
            'new-password',
            showConfirmPassword,
            () => togglePasswordVisibility('confirmPassword'),
            formData.confirmPassword,
            errors.confirmPassword,
          )}

          <div className="grid gap-2">
            <Label htmlFor="role">Perfil</Label>
            <select
              id="role"
              name="role"
              value={formData.role}
              onChange={handleChange}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              aria-invalid={Boolean(errors.role)}
            >
              <option value="client">Cliente</option>
              <option value="employee">Colaborador</option>
            </select>
            {errors.role ? <p className="text-sm text-destructive">{errors.role}</p> : null}
          </div>
        </div>

        <Button type="submit" size="lg" disabled={isLoading} className={styles.submitButton}>
          {isLoading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              A registar…
            </>
          ) : (
            'Criar conta'
          )}
        </Button>

        <AuthShellFooter>
          <p className="text-center text-sm text-muted-foreground">
            Já tem conta?{' '}
            <Button type="button" variant="link" className="h-auto p-0" onClick={() => router.push('/auth/login')}>
              Iniciar sessão
            </Button>
          </p>
        </AuthShellFooter>
      </form>
    </AuthShell>
  );
}
