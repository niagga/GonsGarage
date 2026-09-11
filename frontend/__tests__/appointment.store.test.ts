import { renderHook, act } from '@testing-library/react';
import { useAppointmentStore, useAppointments } from '@/stores/appointment.store';
import { appointmentApi } from '@/lib/api/appointment.api';
import type { Appointment, CreateAppointmentRequest } from '@/types/appointment';

jest.mock('@/lib/api/appointment.api');
const mockedAppointmentApi = appointmentApi as jest.Mocked<typeof appointmentApi>;

const sampleAppointment = (over: Partial<Appointment> = {}): Appointment => ({
  id: '1',
  clientName: 'Jane',
  carId: 'car1',
  service: 'oil_change',
  date: '2024-12-15',
  time: '10:00',
  status: 'scheduled',
  createdAt: '2024-12-01T10:00:00Z',
  updatedAt: '2024-12-01T10:00:00Z',
  ...over,
});

describe('AppointmentStore', () => {
  beforeEach(() => {
    useAppointmentStore.getState().reset();
    jest.clearAllMocks();
  });

  describe('fetchAppointments', () => {
    it('should fetch appointments successfully', async () => {
      const mockAppointments: Appointment[] = [sampleAppointment()];

      mockedAppointmentApi.getAppointments.mockResolvedValueOnce(mockAppointments);

      const { result } = renderHook(() => useAppointments());

      await act(async () => {
        await result.current.fetchAppointments();
      });

      expect(result.current.appointments).toEqual(mockAppointments);
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBeNull();
      expect(mockedAppointmentApi.getAppointments).toHaveBeenCalledTimes(1);
    });

    it('should handle fetch appointments error', async () => {
      const errorMessage = 'Failed to fetch appointments';
      mockedAppointmentApi.getAppointments.mockRejectedValueOnce(new Error(errorMessage));

      const { result } = renderHook(() => useAppointments());

      await act(async () => {
        await result.current.fetchAppointments();
      });

      expect(result.current.appointments).toEqual([]);
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBe(errorMessage);
    });
  });

  describe('createAppointment', () => {
    it('should create appointment successfully', async () => {
      const newAppointmentData: CreateAppointmentRequest = {
        clientName: 'Jane',
        carId: 'car1',
        service: 'oil_change',
        date: '2024-12-15',
        time: '10:00',
        status: 'scheduled',
        notes: 'Regular maintenance',
        createdAt: '2024-12-01T10:00:00Z',
        updatedAt: '2024-12-01T10:00:00Z',
      };

      const createdAppointment: Appointment = {
        id: '1',
        clientName: newAppointmentData.clientName,
        carId: newAppointmentData.carId,
        service: newAppointmentData.service,
        date: newAppointmentData.date,
        time: newAppointmentData.time,
        status: 'scheduled',
        notes: newAppointmentData.notes,
        createdAt: newAppointmentData.createdAt,
        updatedAt: newAppointmentData.updatedAt,
      };

      mockedAppointmentApi.createAppointment.mockResolvedValueOnce(createdAppointment);

      const { result } = renderHook(() => useAppointments());

      let success = false;
      await act(async () => {
        success = await result.current.createAppointment(newAppointmentData);
      });

      expect(success).toBe(true);
      expect(result.current.appointments).toContainEqual(createdAppointment);
      expect(result.current.isCreating).toBe(false);
      expect(result.current.error).toBeNull();
      expect(mockedAppointmentApi.createAppointment).toHaveBeenCalledWith(newAppointmentData);
    });

    it('should handle create appointment error', async () => {
      const newAppointmentData: CreateAppointmentRequest = {
        clientName: 'Jane',
        carId: 'car1',
        service: 'oil_change',
        date: '2024-12-15',
        time: '10:00',
        status: 'scheduled',
        createdAt: '2024-12-01T10:00:00Z',
        updatedAt: '2024-12-01T10:00:00Z',
      };

      const errorMessage = 'Failed to create appointment';
      mockedAppointmentApi.createAppointment.mockRejectedValueOnce(new Error(errorMessage));

      const { result } = renderHook(() => useAppointments());

      let success = true;
      await act(async () => {
        success = await result.current.createAppointment(newAppointmentData);
      });

      expect(success).toBe(false);
      expect(result.current.appointments).toEqual([]);
      expect(result.current.isCreating).toBe(false);
      expect(result.current.error).toBe(errorMessage);
    });
  });

  describe('cancelAppointment', () => {
    it('should cancel appointment successfully', async () => {
      const appointment = sampleAppointment();
      const cancelledAppointment: Appointment = {
        ...appointment,
        status: 'cancelled',
        updatedAt: '2024-12-02T10:00:00Z',
      };

      mockedAppointmentApi.cancelAppointment.mockResolvedValueOnce(cancelledAppointment);

      const { result } = renderHook(() => useAppointments());

      act(() => {
        useAppointmentStore.setState({ appointments: [appointment] });
      });

      let success = false;
      await act(async () => {
        success = await result.current.cancelAppointment('1');
      });

      expect(success).toBe(true);
      expect(result.current.appointments[0].status).toBe('cancelled');
      expect(mockedAppointmentApi.cancelAppointment).toHaveBeenCalledWith('1');
    });
  });

  describe('setFilters', () => {
    it('should update filters correctly', () => {
      const { result } = renderHook(() => useAppointments());

      act(() => {
        result.current.setFilters({ status: 'completed', carId: 'car1' });
      });

      expect(result.current.filters).toEqual({
        status: 'completed',
        carId: 'car1',
      });
    });
  });

  describe('clearError', () => {
    it('should clear error state', () => {
      const { result } = renderHook(() => useAppointments());

      act(() => {
        useAppointmentStore.setState({ error: 'Some error' });
      });

      expect(result.current.error).toBe('Some error');

      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('reset', () => {
    it('should reset store to initial state', () => {
      const { result } = renderHook(() => useAppointments());

      act(() => {
        useAppointmentStore.setState({
          appointments: [sampleAppointment()],
          error: 'Some error',
          isLoading: true,
        });
      });

      act(() => {
        result.current.reset();
      });

      expect(result.current.appointments).toEqual([]);
      expect(result.current.error).toBeNull();
      expect(result.current.isLoading).toBe(false);
      expect(result.current.filters).toEqual({ status: 'all' });
    });
  });
});
