import type { Meta, StoryObj } from '@storybook/react';
import { Card, CardHeader, CardBody, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

const meta: Meta<typeof Card> = {
  title: 'UI/Card',
  component: Card,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Card>;

export const Default: Story = {
  render: () => (
    <Card className="max-w-md">
      <CardHeader title="Default card" description="Surface-elevated, shadow-sm, radius-md." />
      <CardBody>Body content with secondary text color by default.</CardBody>
      <CardFooter>
        <Button size="sm" variant="secondary">Action</Button>
      </CardFooter>
    </Card>
  ),
};

export const Interactive: Story = {
  render: () => (
    <Card interactive className="max-w-md">
      <CardHeader title="Interactive card" description="Hover lifts the shadow and accents the border." />
      <CardBody>Use for clickable tiles and navigation.</CardBody>
    </Card>
  ),
};
