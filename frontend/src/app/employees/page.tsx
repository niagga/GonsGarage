'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import { CreateEmployeeRequest, Employee, apiClient } from '@/lib/api';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { BrandLogo } from '@/components/brand/BrandLogo';

export default function EmployeesPage() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingEmployee, setEditingEmployee] = useState<Employee | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalEmployees, setTotalEmployees] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  
  const employeesPerPage = 10;
  
  const router = useRouter();
  const { user, logout } = useAuth();

  const fetchEmployees = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const offset = (currentPage - 1) * employeesPerPage;
      const { data, error: apiError } = await apiClient.getEmployees(employeesPerPage, offset);
      
      if (data && !apiError) {
        setEmployees(data.employees || []);
        setTotalEmployees(data.total || 0);
      } else {
        setError(apiError?.message || 'Não foi possível carregar os colaboradores');
      }
    } catch {
      setError('Erro de rede. Tente novamente.');
    } finally {
      setLoading(false);
    }
  }, [currentPage, employeesPerPage]);

  useEffect(() => {
    if (!user) {
      router.push('/auth/login');
      return;
    }
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void fetchEmployees();
    });
    return () => {
      cancelled = true;
    };
  }, [user, router, fetchEmployees]);

  const handleDelete = async (id: string) => {
    if (!confirm('Tem a certeza de que pretende eliminar este colaborador?')) {
      return;
    }

    try {
      const { error } = await apiClient.deleteEmployee(id);
      if (!error) {
        setEmployees(employees.filter(emp => emp.id !== id));
        setTotalEmployees(prev => prev - 1);
      } else {
        alert('Não foi possível eliminar o colaborador: ' + error.message);
      }
    } catch {
      alert('Erro de rede. Tente novamente.');
    }
  };

  const filteredEmployees = employees.filter(employee =>
    employee.first_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    employee.last_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    employee.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
    employee.department.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const totalPages = Math.ceil(totalEmployees / employeesPerPage);

  if (loading && employees.length === 0) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A carregar colaboradores" />
      </div>
    );
  }

  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: 'var(--surface-page)',
    }}>
      {/* Header */}
      <header style={{
        backgroundColor: 'var(--surface-header)',
        boxShadow: 'var(--shadow-sm)',
        borderBottom: '1px solid var(--color-gray-200)',
      }}>
        <div style={{
          maxWidth: '1200px',
          margin: '0 auto',
          padding: '0 var(--space-4)',
        }}>
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            height: '64px',
          }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--space-3)',
            }}>
              <div
                style={{
                  width: '32px',
                  height: '32px',
                  borderRadius: 'var(--radius)',
                  overflow: 'hidden',
                  flexShrink: 0,
                }}
              >
                <BrandLogo alt="" width={32} height={32} style={{ objectFit: 'cover', width: '100%', height: '100%' }} />
              </div>
              <div>
                <h1 style={{
                  fontSize: '1.25rem',
                  fontWeight: '700',
                  color: 'var(--color-gray-900)',
                  margin: 0,
                }}>
                  GonsGarage
                </h1>
                <p style={{
                  fontSize: '0.75rem',
                  color: 'var(--color-gray-600)',
                  margin: 0,
                }}>
                  Gestão de colaboradores
                </p>
              </div>
            </div>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--space-3)',
            }}>
              <span style={{
                fontSize: '0.875rem',
                color: 'var(--color-gray-700)',
              }}>
                Olá, {user?.firstName} {user?.lastName}
              </span>
              <Button type="button" variant="destructive" size="sm" onClick={logout}>
                Terminar sessão
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main style={{
        maxWidth: '1200px',
        margin: '0 auto',
        padding: 'var(--space-6) var(--space-4)',
      }}>
        {/* Controls */}
        <div style={{
          marginBottom: 'var(--space-6)',
          display: 'flex',
          flexDirection: 'column',
          gap: 'var(--space-4)',
        }}>
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 'var(--space-4)',
          }}>
            <div style={{
              position: 'relative',
              maxWidth: '320px',
              flex: '1',
            }}>
              <Label htmlFor="employee-search" className="sr-only">
                Pesquisar colaboradores
              </Label>
              <Input
                id="employee-search"
                type="text"
                placeholder="Pesquisar colaboradores…"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-9"
                style={{ width: '100%' }}
              />
              <svg 
                style={{
                  position: 'absolute',
                  left: 'var(--space-2)',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  width: '16px',
                  height: '16px',
                  color: 'var(--color-gray-400)',
                  pointerEvents: 'none',
                }} 
                fill="none" 
                viewBox="0 0 24 24" 
                stroke="currentColor"
                aria-hidden
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
            <Button type="button" onClick={() => setShowCreateModal(true)} className="gap-2">
              <svg 
                style={{ width: '16px', height: '16px' }} 
                fill="none" 
                viewBox="0 0 24 24" 
                stroke="currentColor"
                aria-hidden
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              Adicionar colaborador
            </Button>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div style={{
            marginBottom: 'var(--space-4)',
            backgroundColor: 'var(--chip-danger-bg)',
            border: '1px solid var(--chip-danger-border)',
            color: 'var(--color-error)',
            padding: 'var(--space-3)',
            borderRadius: 'var(--radius)',
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--space-2)',
          }}>
            <svg 
              style={{ width: '16px', height: '16px', flexShrink: 0 }} 
              viewBox="0 0 20 20" 
              fill="currentColor"
            >
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <span style={{ fontSize: '0.875rem' }}>{error}</span>
          </div>
        )}

        {/* Employee Table Container */}
        <div style={{
          backgroundColor: 'var(--surface-header)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-md)',
          overflow: 'hidden',
        }}>
          {/* Table Header */}
          <div style={{
            padding: 'var(--space-4) var(--space-6)',
            borderBottom: '1px solid var(--color-gray-200)',
          }}>
            <h3 style={{
              fontSize: '1.125rem',
              fontWeight: '600',
              color: 'var(--color-gray-900)',
              margin: 0,
              marginBottom: 'var(--space-1)',
            }}>
              Colaboradores ({totalEmployees})
            </h3>
            <p style={{
              fontSize: '0.875rem',
              color: 'var(--color-gray-600)',
              margin: 0,
            }}>
              Gerir a equipa da oficina
            </p>
          </div>
          
          {/* Table */}
          <div style={{ overflowX: 'auto' }}>
            <table style={{
              width: '100%',
              borderCollapse: 'collapse',
            }}>
              <thead style={{
                backgroundColor: 'var(--surface-page)',
              }}>
                <tr>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    Nome
                  </th>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    E-mail
                  </th>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    Departamento
                  </th>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    Cargo
                  </th>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    Salário
                  </th>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    Estado
                  </th>
                  <th style={{
                    padding: 'var(--space-3) var(--space-6)',
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontWeight: '500',
                    color: 'var(--color-gray-500)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}>
                    Ações
                  </th>
                </tr>
              </thead>
              <tbody style={{
                backgroundColor: 'var(--surface-header)',
              }}>
                {filteredEmployees.length === 0 ? (
                  <tr>
                    <td 
                      colSpan={7} 
                      style={{
                        padding: 'var(--space-8) var(--space-6)',
                        textAlign: 'center',
                        color: 'var(--color-gray-500)',
                        fontSize: '0.875rem',
                      }}
                    >
                      {searchTerm ? 'Nenhum colaborador corresponde à pesquisa.' : 'Sem colaboradores. Adicione o primeiro!'}
                    </td>
                  </tr>
                ) : (
                  filteredEmployees.map((employee, index) => (
                    <tr 
                      key={employee.id}
                      style={{
                        borderTop: index > 0 ? '1px solid var(--color-gray-200)' : 'none',
                        transition: 'background-color var(--transition-fast)',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.backgroundColor = 'var(--surface-muted)';
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.backgroundColor = 'var(--surface-header)';
                      }}
                    >
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                      }}>
                        <div style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 'var(--space-3)',
                        }}>
                          <div style={{
                            width: '32px',
                            height: '32px',
                            backgroundColor: 'var(--color-primary)',
                            borderRadius: '50%',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            flexShrink: 0,
                          }}>
                            <span style={{
                              fontSize: '0.75rem',
                              fontWeight: '500',
                              color: 'var(--text-on-primary)',
                            }}>
                              {employee.first_name[0]}{employee.last_name[0]}
                            </span>
                          </div>
                          <div>
                            <div style={{
                              fontSize: '0.875rem',
                              fontWeight: '500',
                              color: 'var(--color-gray-900)',
                            }}>
                              {employee.first_name} {employee.last_name}
                            </div>
                            <div style={{
                              fontSize: '0.75rem',
                              color: 'var(--color-gray-500)',
                            }}>
                              {employee.phone || 'No phone'}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                        fontSize: '0.875rem',
                        color: 'var(--color-gray-900)',
                      }}>
                        {employee.email}
                      </td>
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                        fontSize: '0.875rem',
                        color: 'var(--color-gray-900)',
                      }}>
                        {employee.department}
                      </td>
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                        fontSize: '0.875rem',
                        color: 'var(--color-gray-900)',
                      }}>
                        {employee.position}
                      </td>
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                        fontSize: '0.875rem',
                        color: 'var(--color-gray-900)',
                      }}>
                        ${employee.salary.toLocaleString()}
                      </td>
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                      }}>
                        <span style={{
                          display: 'inline-flex',
                          padding: 'var(--space-1) var(--space-2)',
                          fontSize: '0.75rem',
                          fontWeight: '500',
                          borderRadius: 'var(--radius)',
                          backgroundColor: employee.is_active ? 'var(--chip-success-bg)' : 'var(--chip-danger-bg)',
                          color: employee.is_active ? 'var(--chip-success-fg)' : 'var(--chip-danger-fg)',
                        }}>
                          {employee.is_active ? 'Ativo' : 'Inativo'}
                        </span>
                      </td>
                      <td style={{
                        padding: 'var(--space-4) var(--space-6)',
                        whiteSpace: 'nowrap',
                      }}>
                        <div style={{
                          display: 'flex',
                          gap: 'var(--space-2)',
                        }}>
                          <Button
                            type="button"
                            variant="link"
                            className="h-auto p-0 text-[0.875rem] font-medium text-primary"
                            onClick={() => setEditingEmployee(employee)}
                          >
                            Editar
                          </Button>
                          <Button
                            type="button"
                            variant="link"
                            className="h-auto p-0 text-[0.875rem] font-medium text-destructive"
                            onClick={() => handleDelete(employee.id)}
                          >
                            Eliminar
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div style={{
              backgroundColor: 'var(--surface-header)',
              padding: 'var(--space-3) var(--space-6)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              borderTop: '1px solid var(--color-gray-200)',
            }}>
              <div>
                <p style={{
                  fontSize: '0.875rem',
                  color: 'var(--color-gray-700)',
                  margin: 0,
                }}>
                  A mostrar{' '}
                  <span style={{ fontWeight: '500' }}>{(currentPage - 1) * employeesPerPage + 1}</span>
                  {' '}a{' '}
                  <span style={{ fontWeight: '500' }}>
                    {Math.min(currentPage * employeesPerPage, totalEmployees)}
                  </span>
                  {' '}de{' '}
                  <span style={{ fontWeight: '500' }}>{totalEmployees}</span>
                  {' '}resultados
                </p>
              </div>
              <div style={{
                display: 'flex',
                gap: 'var(--space-1)',
              }}>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
                  disabled={currentPage === 1}
                >
                  Anterior
                </Button>
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  const pageNum = i + 1;
                  return (
                    <Button
                      key={pageNum}
                      type="button"
                      variant={pageNum === currentPage ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setCurrentPage(pageNum)}
                    >
                      {pageNum}
                    </Button>
                  );
                })}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.min(prev + 1, totalPages))}
                  disabled={currentPage === totalPages}
                >
                  Seguinte
                </Button>
              </div>
            </div>
          )}
        </div>
      </main>

      {/* Create/Edit Employee Modal */}
      {(showCreateModal || editingEmployee) && (
        <EmployeeModal
          employee={editingEmployee}
          onClose={() => {
            setShowCreateModal(false);
            setEditingEmployee(null);
          }}
          onSuccess={() => {
            fetchEmployees();
            setShowCreateModal(false);
            setEditingEmployee(null);
          }}
        />
      )}

    </div>
  );
}

// Employee Modal Component with consistent styling
interface EmployeeModalProps {
  employee?: Employee | null;
  onClose: () => void;
  onSuccess: () => void;
}

function EmployeeModal({ employee, onClose, onSuccess }: EmployeeModalProps) {
  const [formData, setFormData] = useState<CreateEmployeeRequest>({
    first_name: employee?.first_name || '',
    last_name: employee?.last_name || '',
    email: employee?.email || '',
    phone: employee?.phone || '',
    department: employee?.department || '',
    position: employee?.position || '',
    hire_date: employee?.hire_date ? employee.hire_date.split('T')[0] : new Date().toISOString().split('T')[0],
    salary: employee?.salary || 0,
  });
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [isLoading, setIsLoading] = useState(false);

  const validateForm = () => {
    const newErrors: { [key: string]: string } = {};

    if (!formData.first_name) newErrors.first_name = 'O nome é obrigatório';
    if (!formData.last_name) newErrors.last_name = 'O apelido é obrigatório';
    if (!formData.email) newErrors.email = 'O e-mail é obrigatório';
    else if (!/\S+@\S+\.\S+/.test(formData.email)) newErrors.email = 'E-mail inválido';
    if (!formData.department) newErrors.department = 'O departamento é obrigatório';
    if (!formData.position) newErrors.position = 'O cargo é obrigatório';
    if (!formData.hire_date) newErrors.hire_date = 'A data de admissão é obrigatória';
    if (!formData.salary || formData.salary <= 0) newErrors.salary = 'O salário tem de ser superior a 0';

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) return;

    setIsLoading(true);
    setErrors({});

    try {
      let result;
      if (employee) {
        result = await apiClient.updateEmployee(employee.id, formData);
      } else {
        result = await apiClient.createEmployee(formData);
      }

      if (result.data && !result.error) {
        onSuccess();
      } else {
        setErrors({ general: result.error?.message || 'Operação falhou' });
      }
    } catch {
      setErrors({ general: 'Erro de rede. Tente novamente.' });
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: name === 'salary' ? Number.parseFloat(value) || 0 : value
    }));
    
    if (errors[name]) {
      setErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const selectFieldClass = cn(
    'flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
  );

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      backgroundColor: 'rgba(0, 0, 0, 0.5)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 'var(--space-4)',
      zIndex: 50,
    }}>
      <div style={{
        backgroundColor: 'var(--surface-header)',
        borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-lg)',
        width: '100%',
        maxWidth: '500px',
        maxHeight: '90vh',
        overflow: 'hidden',
      }}>
        {/* Modal Header */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: 'var(--space-4) var(--space-6)',
          borderBottom: '1px solid var(--color-gray-200)',
        }}>
          <h3 style={{
            fontSize: '1.125rem',
            fontWeight: '600',
            color: 'var(--color-gray-900)',
            margin: 0,
          }}>
            {employee ? 'Editar colaborador' : 'Novo colaborador'}
          </h3>
          <Button type="button" variant="ghost" size="icon" onClick={onClose} aria-label="Fechar">
            <svg style={{ width: '20px', height: '20px' }} fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Button>
        </div>

        {/* Modal Body */}
        <div style={{
          padding: 'var(--space-6)',
          maxHeight: '70vh',
          overflowY: 'auto',
        }}>
          <form onSubmit={handleSubmit}>
            {errors.general && (
              <div style={{
                backgroundColor: 'var(--chip-danger-bg)',
                border: '1px solid var(--chip-danger-border)',
                color: 'var(--color-error)',
                padding: 'var(--space-3)',
                borderRadius: 'var(--radius)',
                marginBottom: 'var(--space-4)',
                fontSize: '0.875rem',
              }}>
                {errors.general}
              </div>
            )}

            <div style={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr',
              gap: 'var(--space-4)',
              marginBottom: 'var(--space-4)',
            }}>
              <div>
                <Label htmlFor="emp-first_name" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                  Nome
                </Label>
                <Input
                  id="emp-first_name"
                  type="text"
                  name="first_name"
                  value={formData.first_name}
                  onChange={handleChange}
                  className={cn(errors.first_name && 'border-destructive')}
                  aria-invalid={!!errors.first_name}
                />
                {errors.first_name && (
                  <p style={{
                    marginTop: 'var(--space-1)',
                    fontSize: '0.75rem',
                    color: 'var(--color-error)',
                  }}>
                    {errors.first_name}
                  </p>
                )}
              </div>

              <div>
                <Label htmlFor="emp-last_name" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                  Apelido
                </Label>
                <Input
                  id="emp-last_name"
                  type="text"
                  name="last_name"
                  value={formData.last_name}
                  onChange={handleChange}
                  className={cn(errors.last_name && 'border-destructive')}
                  aria-invalid={!!errors.last_name}
                />
                {errors.last_name && (
                  <p style={{
                    marginTop: 'var(--space-1)',
                    fontSize: '0.75rem',
                    color: 'var(--color-error)',
                  }}>
                    {errors.last_name}
                  </p>
                )}
              </div>
            </div>

            <div style={{ marginBottom: 'var(--space-4)' }}>
              <Label htmlFor="emp-email" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                E-mail
              </Label>
              <Input
                id="emp-email"
                type="email"
                name="email"
                value={formData.email}
                onChange={handleChange}
                className={cn(errors.email && 'border-destructive')}
                aria-invalid={!!errors.email}
              />
              {errors.email && (
                <p style={{
                  marginTop: 'var(--space-1)',
                  fontSize: '0.75rem',
                  color: 'var(--color-error)',
                }}>
                  {errors.email}
                </p>
              )}
            </div>

            <div style={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr',
              gap: 'var(--space-4)',
              marginBottom: 'var(--space-4)',
            }}>
              <div>
                <Label htmlFor="emp-department" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                  Departamento
                </Label>
                <select
                  id="emp-department"
                  name="department"
                  value={formData.department}
                  onChange={handleChange}
                  className={cn(selectFieldClass, errors.department && 'border-destructive')}
                  aria-invalid={!!errors.department}
                >
                  <option value="">Selecione o departamento</option>
                  <option value="Mechanical">Mecânica</option>
                  <option value="Electrical">Elétrica</option>
                  <option value="Body Work">Chapa e pintura</option>
                  <option value="Administration">Administração</option>
                  <option value="Sales">Vendas</option>
                </select>
                {errors.department && (
                  <p style={{
                    marginTop: 'var(--space-1)',
                    fontSize: '0.75rem',
                    color: 'var(--color-error)',
                  }}>
                    {errors.department}
                  </p>
                )}
              </div>

              <div>
                <Label htmlFor="emp-position" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                  Cargo
                </Label>
                <Input
                  id="emp-position"
                  type="text"
                  name="position"
                  value={formData.position}
                  onChange={handleChange}
                  className={cn(errors.position && 'border-destructive')}
                  aria-invalid={!!errors.position}
                />
                {errors.position && (
                  <p style={{
                    marginTop: 'var(--space-1)',
                    fontSize: '0.75rem',
                    color: 'var(--color-error)',
                  }}>
                    {errors.position}
                  </p>
                )}
              </div>
            </div>

            <div style={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr',
              gap: 'var(--space-4)',
              marginBottom: 'var(--space-4)',
            }}>
              <div>
                <Label htmlFor="emp-hire_date" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                  Data de admissão
                </Label>
                <Input
                  id="emp-hire_date"
                  type="date"
                  name="hire_date"
                  value={formData.hire_date}
                  onChange={handleChange}
                  className={cn(errors.hire_date && 'border-destructive')}
                  aria-invalid={!!errors.hire_date}
                />
                {errors.hire_date && (
                  <p style={{
                    marginTop: 'var(--space-1)',
                    fontSize: '0.75rem',
                    color: 'var(--color-error)',
                  }}>
                    {errors.hire_date}
                  </p>
                )}
              </div>

              <div>
                <Label htmlFor="emp-salary" className="mb-1 block text-[0.875rem] text-[var(--color-gray-700)]">
                  Salário
                </Label>
                <Input
                  id="emp-salary"
                  type="number"
                  name="salary"
                  value={formData.salary}
                  onChange={handleChange}
                  min={0}
                  step="0.01"
                  className={cn(errors.salary && 'border-destructive')}
                  aria-invalid={!!errors.salary}
                />
                {errors.salary && (
                  <p style={{
                    marginTop: 'var(--space-1)',
                    fontSize: '0.75rem',
                    color: 'var(--color-error)',
                  }}>
                    {errors.salary}
                  </p>
                )}
              </div>
            </div>

            <div style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 'var(--space-3)',
              paddingTop: 'var(--space-4)',
              borderTop: '1px solid var(--color-gray-200)',
            }}>
              <Button type="button" variant="outline" onClick={onClose}>
                Cancelar
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'A guardar…' : (employee ? 'Atualizar' : 'Criar')}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}