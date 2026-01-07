# Nx Monorepo Initialization

Set up Nx monorepo with proper workspace structure for Go backend and React frontend. Configure shared TypeScript types, OpenAPI SDK generation, and build pipelines.

## Rationale

Nx provides a robust monorepo structure that enables code sharing between frontend and backend, consistent build tooling, and scalable project organization essential for a unified media management tool.

## User Stories

- As a developer, I want a well-organized monorepo so that I can efficiently develop both frontend and backend

## Acceptance Criteria

- [ ] Nx workspace initialized with apps/ and libs/ directories
- [ ] Go backend app scaffolded under apps/api
- [ ] React frontend app scaffolded under apps/web
- [ ] Shared TypeScript types library created
- [ ] Basic build and dev commands work via nx run
