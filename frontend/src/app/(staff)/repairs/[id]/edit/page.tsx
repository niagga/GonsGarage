
"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { RepairForm } from "@/components/repairs/RepairForm";
import { apiClient, Repair } from "@/lib/api";
import { toast } from "@/components/ui/use-toast";

export default function EditRepairPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const [repair, setRepair] = useState<Repair | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    const fetchRepair = async () => {
      const response = await apiClient.getRepair(params.id);
      if (response.data) {
        setRepair(response.data);
      } else {
        toast({ title: "Error fetching repair", description: response.error?.message, variant: "destructive" });
        router.back();
      }
      setIsLoading(false);
    };
    fetchRepair();
  }, [params.id, router]);

  const onSubmit = async (values: any) => {
    setIsSubmitting(true);
    const response = await apiClient.updateRepair(params.id, values);
    setIsSubmitting(false);

    if (response.data) {
      toast({ title: "Repair updated successfully" });
      router.push(`/repairs/${params.id}`);
    } else {
      toast({ title: "Error updating repair", description: response.error?.message, variant: "destructive" });
    }
  };

  if (isLoading) return <div>Loading...</div>;
  if (!repair) return null;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Edit Repair</h1>
      <RepairForm 
        initialValues={{
          car_id: repair.car_id,
          description: repair.description,
          status: repair.status,
          cost: repair.cost,
          started_at: repair.started_at,
        }} 
        onSubmit={onSubmit} 
        isSubmitting={isSubmitting} 
        isEdit={true}
      />
    </div>
  );
}
