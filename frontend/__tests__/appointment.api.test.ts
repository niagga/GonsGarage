import { appointmentApi } from '@/lib/api/appointment.api';
import { apiClient } from '@/lib/api-client';
import type { Appointment, CreateAppointmentRequest } from '@/types/appointment';

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockedClient = apiClient as jest.Mocked<typeof apiClient>;

describe('appointmentApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('getAppointments returns list when API succeeds', async () => {
    const rows: Appointment[] = [
      {
        id: '1',
        clientName: 'Jane',
        carId: 'car1',
        service: 'oil_change',
        date: '2024-12-15',
        time: '10:00',
        status: 'scheduled',
        createdAt: '2024-12-01T10:00:00Z',
        updatedAt: '2024-12-01T10:00:00Z',
      },
    ];
    mockedClient.get.mockResolvedValueOnce({ success: true, data: rows });

    const result = await appointmentApi.getAppointments();

    expect(result).toEqual(rows);
    expect(mockedClient.get).toHaveBeenCalledWith('/appointments');
  });

  it('getAppointments returns empty array when API reports failure', async () => {
    mockedClient.get.mockResolvedValueOnce({
      success: false,
      error: { message: 'nope', status: 500 },
    });

    const result = await appointmentApi.getAppointments();

    expect(result).toEqual([]);
  });

  it('createAppointment sends camelCase body and returns entity', async () => {
    const req: CreateAppointmentRequest = {
      clientName: 'Jane',
      carId: 'car1',
      service: 'brake',
      date: '2024-12-16',
      time: '11:00',
      status: 'scheduled',
      notes: 'check pads',
      createdAt: '2024-12-01T10:00:00Z',
      updatedAt: '2024-12-01T10:00:00Z',
    };
    const created: Appointment = {
      id: '99',
      clientName: req.clientName,
      carId: req.carId,
      service: req.service,
      date: req.date,
      time: req.time,
      status: req.status,
      notes: req.notes,
      createdAt: req.createdAt,
      updatedAt: req.updatedAt,
    };
    mockedClient.post.mockResolvedValueOnce({ success: true, data: created });

    const result = await appointmentApi.createAppointment(req);

    expect(result).toEqual(created);
    expect(mockedClient.post).toHaveBeenCalledWith('/appointments', {
      carId: 'car1',
      serviceType: 'brake',
      scheduledAt: '2024-12-16',
      notes: 'check pads',
      status: 'scheduled',
    });
  });
});
