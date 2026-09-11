import { fetchMyInvoiceDetailAuthenticated } from '@/lib/server/my-invoices-initial';
import MyInvoiceDetailClient from './MyInvoiceDetailClient';

type PageProps = Readonly<{
  params: Promise<{ id: string }>;
}>;

export default async function MyInvoiceDetailPage({ params }: PageProps) {
  const { id } = await params;
  const initialRow = await fetchMyInvoiceDetailAuthenticated(id);
  return <MyInvoiceDetailClient invoiceId={id} initialRow={initialRow} />;
}
