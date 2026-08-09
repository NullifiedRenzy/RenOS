import type { Meta, StoryObj } from '@storybook/react';
import { Badge } from '@/components/ui/badge';

const meta: Meta<typeof Badge> = {
  title: 'UI/Badge',
  component: Badge,
  tags: ['autodocs'],
  argTypes: {
    tone: { control: 'select', options: ['neutral', 'success', 'warning', 'error', 'info', 'accent'] },
  },
};

export default meta;
type Story = StoryObj<typeof Badge>;

export const AllTones: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge tone="neutral">neutral</Badge>
      <Badge tone="success" dot>online</Badge>
      <Badge tone="warning" dot>degraded</Badge>
      <Badge tone="error" dot>critical</Badge>
      <Badge tone="info" dot>info</Badge>
      <Badge tone="accent" dot>accent</Badge>
    </div>
  ),
};

export const WithDot: Story = {
  args: { tone: 'success', dot: true, children: 'online' },
};
