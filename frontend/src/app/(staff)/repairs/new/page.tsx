
"use client";

import { useRouter } from "next/navigation";
import { RepairForm } from "@/components/repairs/RepairForm";
import { apiClient } from "@/lib/api";
import { useState } from "react";
import { toast } from "@/components/ui/use-toast"; // Assuming toast is available

export default function NewRepairPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const onSubmit = async (values: any) => {
    setIsSubmitting(true);
    const response = await apiClient.createRepair(values);
    setIsSubmitting(false);

    if (response.data) {
      toast({ title: "Repair created successfully" });
      router.push(`/repairs/${response.data.id}`);
    } else {
      toast({ title: "Error creating repair", description: response.error?.message, variant: "destructive" });
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Create Repair</h1>
      <RepairForm onSubmit={onSubmit} isSubmitting={isSubmitting} />
    </div>
  );
}
