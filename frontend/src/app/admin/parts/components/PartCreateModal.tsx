'use client';

import React, { useEffect, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import type { PartItemWriteBody, PartUOM } from '@/types/parts';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { PART_UOM_OPTIONS } from '../part-uom-options';

export interface PartCreateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: (partId: string) => void;
}

export function PartCreateModal({ open, onOpenChange, onSuccess }: Readonly<PartCreateModalProps>) {
  const [reference, setReference] = useState('');
  const [brand, setBrand] = useState('');
  const [name, setName] = useState('');
  const [barcode, setBarcode] = useState('');
  const [quantity, setQuantity] = useState('0');
  const [uom, setUom] = useState<PartUOM>('unit');
  const [minimumQuantity, setMinimumQuantity] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setReference('');
      setBrand('');
      setName('');
      setBarcode('');
      setQuantity('0');
      setUom('unit');
      setMinimumQuantity('');
      setFormError(null);
      setSaving(false);
    });
    return () => {
      cancelled = true;
    };
  }, [open]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFormError(null);
    const qty = Number(quantity);
    const body: PartItemWriteBody = {
      reference: reference.trim(),
      brand: brand.trim(),
      name: name.trim(),
      barcode: barcode.trim(),
      quantity: Number.isFinite(qty) ? qty : 0,
      uom,
    };
    const min = minimumQuantity.trim();
    if (min !== '') {
      const m = Number.parseFloat(min);
      body.minimumQuantity = Number.isFinite(m) ? m : null;
    }
    const res = await apiClient.createPart(body);
    setSaving(false);
    if (res.success && res.data) {
      onSuccess(res.data.id);
      return;
    }
    setFormError(res.error?.message ?? 'Não foi possível criar a peça.');
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg" aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>Nova peça</DialogTitle>
        </DialogHeader>
        {formError ? <p className="text-sm text-destructive">{formError}</p> : null}
        <form className="grid gap-3 pt-1" onSubmit={(ev) => void onSubmit(ev)}>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-reference">Referência</Label>
            <Input
              id="part-modal-reference"
              value={reference}
              onChange={ev => setReference(ev.target.value)}
              required
              autoComplete="off"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-brand">Marca</Label>
            <Input id="part-modal-brand" value={brand} onChange={ev => setBrand(ev.target.value)} required autoComplete="off" />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-name">Nome</Label>
            <Input id="part-modal-name" value={name} onChange={ev => setName(ev.target.value)} required autoComplete="off" />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-barcode">Código de barras</Label>
            <Input id="part-modal-barcode" value={barcode} onChange={ev => setBarcode(ev.target.value)} autoComplete="off" />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-quantity">Quantidade</Label>
            <Input
              id="part-modal-quantity"
              type="number"
              min={0}
              step="any"
              value={quantity}
              onChange={ev => setQuantity(ev.target.value)}
              required
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-uom">Unidade de medida</Label>
            <Select value={uom} onValueChange={value => setUom(value as PartUOM)}>
              <SelectTrigger id="part-modal-uom">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PART_UOM_OPTIONS.map(o => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="part-modal-min">Quantidade mínima (opcional)</Label>
            <Input
              id="part-modal-min"
              type="number"
              min={0}
              step="any"
              value={minimumQuantity}
              onChange={ev => setMinimumQuantity(ev.target.value)}
            />
          </div>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
              Cancelar
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? 'A guardar…' : 'Criar'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
