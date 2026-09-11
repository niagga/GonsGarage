
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Repair } from "@/lib/api";

const repairSchema = z.object({
  car_id: z.string().min(1, "Car ID is required"),
  description: z.string().min(1, "Description is required"),
  status: z.enum(['pending', 'in_progress', 'completed', 'cancelled']),
  cost: z.coerce.number().min(0, "Cost must be a positive number"),
  started_at: z.string().optional(),
});

type RepairFormValues = z.infer<typeof repairSchema>;

interface RepairFormProps {
  initialValues?: Partial<RepairFormValues>;
  onSubmit: (values: RepairFormValues) => void;
  isSubmitting?: boolean;
  isEdit?: boolean;
}

export function RepairForm({ initialValues, onSubmit, isSubmitting, isEdit }: RepairFormProps) {
  const form = useForm<RepairFormValues>({
    resolver: zodResolver(repairSchema),
    defaultValues: initialValues || {
      car_id: "",
      description: "",
      status: 'pending',
      cost: 0,
      started_at: "",
    },
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="car_id"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Car ID</FormLabel>
              <FormControl>
                <Input placeholder="Enter Car ID" {...field} disabled={isEdit} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="description"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Description</FormLabel>
              <FormControl>
                <Input placeholder="Description of the repair" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="status"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Status</FormLabel>
              <Select onValueChange={field.onChange} defaultValue={field.value}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a status" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="pending">Pending</SelectItem>
                  <SelectItem value="in_progress">In Progress</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="cancelled">Cancelled</SelectItem>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="cost"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Cost</FormLabel>
              <FormControl>
                <Input type="number" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "Submitting..." : "Submit"}
        </Button>
      </form>
    </Form>
  );
}
