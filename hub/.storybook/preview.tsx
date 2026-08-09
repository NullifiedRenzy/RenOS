import type { Preview } from '@storybook/nextjs-vite'
import '../styles/globals.css'

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
       color: /(background|color)$/i,
       date: /Date$/i,
      },
    },
    backgrounds: {
      values: [
        { name: 'light', value: '#FFFFFF' },
        { name: 'dark', value: '#1C1C1E' },
      ],
    },
  },
};

export default preview;