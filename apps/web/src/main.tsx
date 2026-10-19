// Bundled fonts (DESIGN.md §Typography). Imported here, not from styles.css, so
// Vite rewrites the woff2 URLs into hashed /assets/ files that ship inside the
// image — Vido never fetches a font from the internet (NAS may be offline).
// Upright variable files only: one set of woff2 covers every weight we use.
import '@fontsource-variable/jetbrains-mono/wght.css';
import '@fontsource-variable/noto-sans-tc/wght.css';
import { StrictMode } from 'react';
import * as ReactDOM from 'react-dom/client';
import { RouterProvider } from '@tanstack/react-router';
import { QueryClientProvider } from '@tanstack/react-query';
import { router } from './router';
import { queryClient } from './queryClient';

const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);

root.render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>
);
