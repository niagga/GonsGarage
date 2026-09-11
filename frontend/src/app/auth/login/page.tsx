import { Suspense } from 'react';
import LoginForm from './LoginForm';

export default function LoginPage() {
  return (
    <Suspense fallback={<div>A carregar…</div>}>
      <LoginForm />
    </Suspense>
  );
}
