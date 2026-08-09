import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Input>;

export const Default: Story = {
  args: { label: 'Hostname', placeholder: 'webserver-01' },
};

export const WithError: Story = {
  args: { label: 'API token', error: 'Token is required', placeholder: 'sk-…' },
};

export const WithHint: Story = {
  args: { label: 'Port', hint: 'Defaults to 22', placeholder: '22' },
};

export const Disabled: Story = {
  args: { label: 'Read only', value: 'locked', disabled: true },
};

export const SelectExample: StoryObj = {
  name: 'Select',
  render: () => (
    <Select
      label="Region"
      options={[
        { value: 'us-east-1', label: 'US East (N. Virginia)' },
        { value: 'eu-west-1', label: 'EU West (Ireland)' },
        { value: 'ap-south-1', label: 'Asia Pacific (Mumbai)' },
      ]}
    />
  ),
};

export const Controlled: Story = {
  render: (args) => {
    const [value, setValue] = useState('');
    return <Input {...args} value={value} onChange={(e) => setValue(e.target.value)} placeholder="Type to see it update…" />;
  },
};
