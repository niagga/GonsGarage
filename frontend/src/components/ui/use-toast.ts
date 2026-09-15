/** Minimal toast helper until a full Toaster UI is added (e.g. sonner). */
export type ToastProps = {
  title?: string;
  description?: string;
  variant?: "default" | "destructive";
};

export function toast({ title, description, variant }: ToastProps) {
  const message = [title, description].filter(Boolean).join(": ");
  if (!message) return;

  if (typeof window !== "undefined") {
    if (variant === "destructive") {
      window.alert(message);
    } else {
      console.info(message);
    }
    return;
  }

  if (variant === "destructive") {
    console.error(message);
  } else {
    console.info(message);
  }
}

export function useToast() {
  return { toast };
}
