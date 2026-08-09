import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Tabs } from '@/components/ui/tabs';

const meta: Meta<typeof Tabs> = {
  title: 'UI/Tabs',
  component: Tabs,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Tabs>;

export const Default: Story = {
  render: () => {
    const [active, setActive] = useState('overview');
    return (
      <Tabs
        tabs={[
          { id: 'overview', label: 'Overview' },
          { id: 'alerts', label: 'Alerts', badge: 3 },
          { id: 'logs', label: 'Logs' },
        ]}
        active={active}
        onChange={setActive}
      />
    );
  },
};

export const WithBadges: Story = {
  render: () => {
    const [active, setActive] = useState('all');
    return (
      <Tabs
        tabs={[
          { id: 'all', label: 'All', badge: 12 },
          { id: 'unread', label: 'Unread', badge: 4 },
          { id: 'archived', label: 'Archived' },
        ]}
        active={active}
        onChange={setActive}
      />
    );
  },
};
