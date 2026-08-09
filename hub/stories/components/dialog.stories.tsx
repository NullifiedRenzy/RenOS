import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Dialog } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

const meta: Meta<typeof Dialog> = {
  title: 'UI/Dialog',
  component: Dialog,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Dialog>;

export const Controlled: Story = {
  render: () => {
    const [open, setOpen] = useState(true);
    return (
      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title="Confirm action"
        description="Focus is trapped, Esc closes, and focus returns to the trigger."
        footer={
          <>
            <Button variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button onClick={() => setOpen(false)}>Confirm</Button>
          </>
        }
      >
        <p className="text-body text-[var(--color-text-secondary)]">
          Press Tab to cycle focus within the dialog. Press Esc to close.
        </p>
      </Dialog>
    );
  },
};
