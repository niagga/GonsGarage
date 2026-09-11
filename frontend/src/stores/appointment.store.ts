/**
 * Client-only: appointment domain state (Zustand + Immer). Do not import from RSC / server files.
 */
// ✅ Appointment Store - Zustand + Immer + TypeScript following Agent.md patterns
// Provides centralized appointment state management with CRUD operations

import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { Appointment, CreateAppointmentRequest, UpdateAppointmentRequest } from '@/types/appointment';
import { appointmentApi } from '@/lib/api/appointment.api';

// ✅ Appointment state interface following Agent.md conventions
interface AppointmentState {
  // Data state
  appointments: Appointment[];
  selectedAppointment: Appointment | null;
  
  // Loading and error states
  isLoading: boolean;
  isCreating: boolean;
  isUpdating: boolean;
  isDeleting: boolean;
  error: string | null;
  
  // Filtering
  filters: {
    status?: Appointment['status'] | 'all';
    serviceType?: string;
    carId?: string;
  };
}

interface AppointmentActions {
  fetchAppointments: () => Promise<void>;
  fetchAppointmentById: (id: string) => Promise<void>;
  createAppointment: (data: CreateAppointmentRequest) => Promise<boolean>;
  updateAppointment: (id: string, data: UpdateAppointmentRequest) => Promise<boolean>;
  cancelAppointment: (id: string) => Promise<boolean>;
  confirmAppointment: (id: string) => Promise<boolean>;
  completeAppointment: (id: string) => Promise<boolean>;
  deleteAppointment: (id: string) => Promise<boolean>;
  selectAppointment: (appointment: Appointment | null) => void;
  setFilters: (filters: Partial<AppointmentState['filters']>) => void;
  clearError: () => void;
  reset: () => void;
}

type AppointmentStore = AppointmentState & AppointmentActions;

// ✅ Initial state following Agent.md conventions
const initialState: AppointmentState = {
  appointments: [],
  selectedAppointment: null,
  isLoading: false,
  isCreating: false,
  isUpdating: false,
  isDeleting: false,
  error: null,
  filters: { status: 'all' },
};

// ✅ Appointment store implementation with Zustand + Immer
export const useAppointmentStore = create<AppointmentStore>()(
  immer((set, get) => ({
    // Initial state
    ...initialState,
    
    // ✅ Fetch appointments
    fetchAppointments: async () => {
      set((state) => {
        state.isLoading = true;
        state.error = null;
      });
      
      try {
        const appointments = await appointmentApi.getAppointments();
        set((state) => {
          state.appointments = appointments;
          state.isLoading = false;
          
          // ✅ Log for debugging
          console.log(`✅ Loaded ${appointments.length} appointments`);
        });
      } catch (error) {
        console.error('❌ Unexpected error in fetchAppointments:', error);
        set((state) => {
          state.appointments = []; // ✅ Set empty array instead of leaving undefined
          state.error = (error as Error).message;
          state.isLoading = false;
        });
      }
    },
    
    // ✅ Fetch single appointment
    fetchAppointmentById: async (id: string) => {
      set((state) => {
        state.isLoading = true;
        state.error = null;
      });
      
      try {
        const appointment = await appointmentApi.getAppointment(id);
        set((state) => {
          state.selectedAppointment = appointment;
          state.isLoading = false;
        });
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
          state.isLoading = false;
        });
      }
    },
    
    // ✅ Create appointment
    createAppointment: async (data: CreateAppointmentRequest): Promise<boolean> => {
      set((state) => {
        state.isCreating = true;
        state.error = null;
      });
      
      try {
        const newAppointment = await appointmentApi.createAppointment(data);
        if(!newAppointment) {
          throw new Error('Não foi possível criar a marcação');
        }

        set((state) => {
          state.appointments.push(newAppointment);
          state.isCreating = false;
        });
        return true;
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
          state.isCreating = false;
        });
        return false;
      }
    },
    
    // ✅ Update appointment
    updateAppointment: async (id: string, data: UpdateAppointmentRequest): Promise<boolean> => {
      set((state) => {
        state.isUpdating = true;
        state.error = null;
      });
      
      try {
        const updatedAppointment = await appointmentApi.updateAppointment(id, data);
        if(!updatedAppointment) {
          throw new Error('Não foi possível atualizar a marcação');
        }

        set((state) => {
          const index = state.appointments.findIndex(apt => apt.id === id);
          if (index !== -1) {
            state.appointments[index] = updatedAppointment;
          }
          if (state.selectedAppointment?.id === id) {
            state.selectedAppointment = updatedAppointment;
          }
          state.isUpdating = false;
        });
        return true;
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
          state.isUpdating = false;
        });
        return false;
      }
    },
    
    // ✅ Cancel appointment
    cancelAppointment: async (id: string): Promise<boolean> => {
      try {
        const cancelledAppointment = await appointmentApi.cancelAppointment(id);
        if (!cancelledAppointment) {
          throw new Error('Não foi possível cancelar a marcação');
        }

        set((state) => {
          const index = state.appointments.findIndex(apt => apt.id === id);
          if (index !== -1) {
            state.appointments[index] = cancelledAppointment;
          }
        });
        return true;
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
        });
        return false;
      }
    },
    
    // ✅ Confirm appointment
    confirmAppointment: async (id: string): Promise<boolean> => {
      try {
        const confirmedAppointment = await appointmentApi.confirmAppointment(id);
        if (!confirmedAppointment) {
          throw new Error('Não foi possível confirmar a marcação');
        }

        set((state) => {
          const index = state.appointments.findIndex(apt => apt.id === id);
          if (index !== -1) {
            state.appointments[index] = confirmedAppointment;
          }
        });
        return true;
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
        });
        return false;
      }
    },
    
    // ✅ Complete appointment
    completeAppointment: async (id: string): Promise<boolean> => {
      try {
        const completedAppointment = await appointmentApi.completeAppointment(id);
        if (!completedAppointment) {
          throw new Error('Não foi possível concluir a marcação');
        }

        set((state) => {
          const index = state.appointments.findIndex(apt => apt.id === id);
          if (index !== -1) {
            state.appointments[index] = completedAppointment;
          }
        });
        return true;
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
        });
        return false;
      }
    },
    
    // ✅ Delete appointment
    deleteAppointment: async (id: string): Promise<boolean> => {
      set((state) => {
        state.isDeleting = true;
        state.error = null;
      });
      
      try {
        await appointmentApi.deleteAppointment(id);
        set((state) => {
          state.appointments = state.appointments.filter(apt => apt.id !== id);
          if (state.selectedAppointment?.id === id) {
            state.selectedAppointment = null;
          }
          state.isDeleting = false;
        });
        return true;
      } catch (error) {
        set((state) => {
          state.error = (error as Error).message;
          state.isDeleting = false;
        });
        return false;
      }
    },
    
    // ✅ Select appointment
    selectAppointment: (appointment: Appointment | null) => {
      set((state) => {
        state.selectedAppointment = appointment;
      });
    },
    
    // ✅ Set filters
    setFilters: (filters: Partial<AppointmentState['filters']>) => {
      set((state) => {
        state.filters = { ...state.filters, ...filters };
      });
    },
    
    // ✅ Clear error
    clearError: () => {
      set((state) => {
        state.error = null;
      });
    },
    
    // ✅ Reset store
    reset: () => {
      set(() => initialState);
    },
  }))
);

// ✅ Convenience hooks following Agent.md patterns
export const useAppointments = () => {
  const store = useAppointmentStore();
  return {
    // Data
    appointments: store.appointments,
    selectedAppointment: store.selectedAppointment,
    filters: store.filters,
    
    // Loading states
    isLoading: store.isLoading,
    isCreating: store.isCreating,
    isUpdating: store.isUpdating,
    isDeleting: store.isDeleting,
    error: store.error,
    
    // Actions
    fetchAppointments: store.fetchAppointments,
    fetchAppointmentById: store.fetchAppointmentById,
    createAppointment: store.createAppointment,
    updateAppointment: store.updateAppointment,
    cancelAppointment: store.cancelAppointment,
    confirmAppointment: store.confirmAppointment,
    completeAppointment: store.completeAppointment,
    deleteAppointment: store.deleteAppointment,
    selectAppointment: store.selectAppointment,
    setFilters: store.setFilters,
    clearError: store.clearError,
    reset: store.reset,

    /*fetchCars: (filters?: CarState['filters']) => Promise<void>;
      fetchCarById: (id: string) => Promise<void>;
      createCar: (carData: CreateCarRequest) => Promise<boolean>;
      updateCar: (id: string, carData: Partial<CreateCarRequest>) => Promise<boolean>;
      deleteCar: (id: string) => Promise<boolean>;
      selectCar: (car: Car | null) => void;
      setFilters: (filters: Partial<CarState['filters']>) => void;
      setPage: (page: number) => void;
      clearError: () => void;
      reset: () => void; */
  };
};