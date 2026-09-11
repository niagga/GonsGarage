import MyInvoicesAuthGate from './MyInvoicesAuthGate';

export default function MyInvoicesLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <MyInvoicesAuthGate>{children}</MyInvoicesAuthGate>;
}
