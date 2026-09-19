import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import RegisterPage from './page';

vi.mock('next/image', () => ({
  default: (props: { src: string; alt: string }) => (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={props.src} alt={props.alt} data-testid="brand-logo" />
  ),
}));

const { mockRegister, mockPush, mockReplace, authState } = vi.hoisted(() => ({
  mockRegister: vi.fn().mockResolvedValue({ success: false }),
  mockPush: vi.fn(),
  mockReplace: vi.fn(),
  authState: { isAuthenticated: false },
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
}));

vi.mock('@/stores', () => ({
  useAuth: () => ({
    register: mockRegister,
    isAuthenticated: authState.isAuthenticated,
  }),
}));

beforeEach(() => {
  mockRegister.mockReset();
  mockRegister.mockResolvedValue({ success: false });
  mockPush.mockReset();
  mockReplace.mockReset();
  authState.isAuthenticated = false;
});

describe('RegisterPage', () => {
  it('uses the shared auth shell title and subtitle', () => {
    render(<RegisterPage />);
    expect(screen.getByRole('heading', { level: 1, name: 'Criar conta' })).toBeInTheDocument();
    expect(screen.getByText('Junte-se à equipa GonsGarage')).toBeInTheDocument();
    expect(screen.getByTestId('brand-logo')).toHaveAttribute('src', '/images/LogoGonsGarage.jpg');
  });

  it('shows the cross-link to login as a button', () => {
    render(<RegisterPage />);
    expect(screen.getByRole('button', { name: 'Iniciar sessão' })).toBeInTheDocument();
  });

  it('shows email field label and placeholder in Portuguese aligned with login', () => {
    render(<RegisterPage />);
    expect(screen.getByLabelText('E-mail')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('O seu e-mail')).toBeInTheDocument();
  });

  it('shows confirm password label in Portuguese', () => {
    render(<RegisterPage />);
    expect(screen.getByLabelText('Confirmar palavra-passe')).toBeInTheDocument();
  });

  it('renders Perfil as a system combobox, not a native select', () => {
    render(<RegisterPage />);
    expect(screen.getByRole('combobox', { name: 'Perfil' })).toBeInTheDocument();
    expect(document.querySelector('select#role')).toBeNull();
  });

  it('does not prefix unexpected errors with English "Error:"', async () => {
    const user = userEvent.setup();
    mockRegister.mockRejectedValueOnce(new Error('Falha de rede'));
    render(<RegisterPage />);

    await user.type(screen.getByLabelText('Nome'), 'Maria');
    await user.type(screen.getByLabelText('Apelido'), 'Silva');
    await user.type(screen.getByLabelText('E-mail'), 'maria@example.com');
    await user.type(screen.getByLabelText('Palavra-passe'), 'secret12');
    await user.type(screen.getByLabelText('Confirmar palavra-passe'), 'secret12');
    await user.click(screen.getByRole('button', { name: 'Criar conta' }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Falha de rede');
    expect(alert.textContent).not.toMatch(/^Error:/);
    await waitFor(() => expect(mockRegister).toHaveBeenCalled());
  });

  it('redirects an already authenticated visitor to /dashboard', async () => {
    authState.isAuthenticated = true;
    render(<RegisterPage />);
    await waitFor(() => {
      expect(mockReplace).toHaveBeenCalledWith('/dashboard');
    });
    expect(mockPush).not.toHaveBeenCalledWith('/employees');
  });
});
