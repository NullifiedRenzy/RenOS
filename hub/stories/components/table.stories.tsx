import type { Meta, StoryObj } from '@storybook/react';
import { Badge } from '@/components/ui/badge';
import { Table } from '@/components/ui/table';

const meta: Meta<typeof Table> = {
  title: 'UI/Table',
  component: Table,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Table>;

type Row = { name: string; status: string; region: string };

const rows: Row[] = [
  { name: 'webserver-01', status: 'online', region: 'us-east-1' },
  { name: 'db-02', status: 'degraded', region: 'eu-west-1' },
  { name: 'cache-01', status: 'offline', region: 'ap-south-1' },
];

export const Default: Story = {
  render: () => (
    <div className="max-w-2xl">
      <Table
        columns={[
          { key: 'name', header: 'Name', cell: (r: Row) => r.name },
          {
            key: 'status',
            header: 'Status',
            cell: (r: Row) => (
              <Badge tone={r.status === 'online' ? 'success' : r.status === 'degraded' ? 'warning' : 'error'} dot>
                {r.status}
              </Badge>
            ),
          },
          { key: 'region', header: 'Region', cell: (r: Row) => r.region },
        ]}
        rows={rows}
        rowKey={(r) => r.name}
      />
    </div>
  ),
};

export const Empty: Story = {
  render: () => (
    <div className="max-w-2xl">
      <Table
        columns={[{ key: 'name', header: 'Name', cell: (r: Row) => r.name }]}
        rows={[]}
        rowKey={(r) => r.name}
        emptyMessage="No devices match the current filter"
      />
    </div>
  ),
};
