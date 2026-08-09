import type { Meta, StoryObj } from '@storybook/react';
import { Skeleton, SkeletonCircle } from '@/components/ui/skeleton';

const meta: Meta<typeof Skeleton> = {
  title: 'UI/Skeleton',
  component: Skeleton,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Skeleton>;

export const SingleLine: Story = {
  args: {},
};

export const MultiLine: Story = {
  render: () => (
    <div className="w-72">
      <Skeleton lines={4} />
    </div>
  ),
};

export const WithAvatar: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <SkeletonCircle size={48} />
      <div className="w-48">
        <Skeleton lines={2} />
      </div>
    </div>
  ),
};
