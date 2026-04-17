import { QueryClient } from '@tanstack/react-query';

// Shared singleton so route loaders (e.g. routes/index.tsx prefetch) and the
// React provider tree both see the same TanStack Query cache.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,
      retry: 1,
    },
  },
});
