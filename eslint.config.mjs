import js from '@eslint/js';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import prettier from 'eslint-config-prettier';
// Local custom rules — CJS plugins, default-imported as their module.exports.
// Each rule lives in its own file; they're merged here under one `local` plugin
// namespace so a single `plugins: { local: localRules }` block surfaces every
// `local/{rule-name}`. Merging inline (vs going through an index.js entry)
// keeps the rule function references identity-equal to ESM consumers (spec
// files), so `expect(...).toBe(rule)` checks pass.
//
// Story 19-3 added `local/implements-pen-node-id` (Rule 21); story 19-9 added
// `local/time-dependent-fixture-stability` (Rule 23).
import implementsPenNodeId from './apps/web/src/eslint-rules/implements-pen-node-id.js';
import timeDependentFixtureStability from './apps/web/src/eslint-rules/time-dependent-fixture-stability.js';

const localRules = {
  rules: {
    ...implementsPenNodeId.rules,
    ...timeDependentFixtureStability.rules,
  },
};

export default [
  // Global ignores
  {
    ignores: [
      'node_modules/**',
      'dist/**',
      'build/**',
      '.nx/**',
      '.turbo/**',
      '*.log',
      'coverage/**',
      '.next/**',
      'out/**',
      '*.config.js',
      '*.config.ts',
      'playwright.config.ts',
      'vite.config.ts',
      'apps/**/vite.config.ts',
      'test-results/**',
      'playwright-report/**',
      '.worktrees/**',
    ],
  },

  // Base JS recommended rules
  js.configs.recommended,

  // TypeScript files
  {
    files: ['**/*.ts', '**/*.tsx'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 2020,
        sourceType: 'module',
        ecmaFeatures: {
          jsx: true,
        },
      },
      globals: {
        // Browser globals
        window: 'readonly',
        document: 'readonly',
        navigator: 'readonly',
        console: 'readonly',
        setTimeout: 'readonly',
        clearTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
        fetch: 'readonly',
        URL: 'readonly',
        URLSearchParams: 'readonly',
        FormData: 'readonly',
        Blob: 'readonly',
        File: 'readonly',
        FileReader: 'readonly',
        localStorage: 'readonly',
        sessionStorage: 'readonly',
        // DOM types (used in TypeScript type annotations)
        React: 'readonly',
        JSX: 'readonly',
        HTMLElement: 'readonly',
        HTMLInputElement: 'readonly',
        HTMLButtonElement: 'readonly',
        HTMLDivElement: 'readonly',
        HTMLFormElement: 'readonly',
        HTMLImageElement: 'readonly',
        HTMLAnchorElement: 'readonly',
        HTMLTextAreaElement: 'readonly',
        HTMLSelectElement: 'readonly',
        KeyboardEvent: 'readonly',
        MouseEvent: 'readonly',
        Event: 'readonly',
        CustomEvent: 'readonly',
        EventTarget: 'readonly',
        Element: 'readonly',
        Node: 'readonly',
        NodeList: 'readonly',
        RequestInit: 'readonly',
        Response: 'readonly',
        Request: 'readonly',
        Headers: 'readonly',
        AbortController: 'readonly',
        AbortSignal: 'readonly',
        EventSource: 'readonly',
        MessageEvent: 'readonly',
        IntersectionObserver: 'readonly',
        IntersectionObserverEntry: 'readonly',
        // Node globals
        process: 'readonly',
        __dirname: 'readonly',
        __filename: 'readonly',
        module: 'readonly',
        require: 'readonly',
        Buffer: 'readonly',
        global: 'readonly',
        // Test globals (Vitest/Jest)
        describe: 'readonly',
        it: 'readonly',
        test: 'readonly',
        expect: 'readonly',
        beforeEach: 'readonly',
        afterEach: 'readonly',
        beforeAll: 'readonly',
        afterAll: 'readonly',
        vi: 'readonly',
        jest: 'readonly',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
      react: react,
      'react-hooks': reactHooks,
    },
    settings: {
      react: {
        version: 'detect',
      },
    },
    rules: {
      // TypeScript rules
      ...tseslint.configs.recommended.rules,
      '@typescript-eslint/explicit-module-boundary-types': 'off',
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
      // Disable base rule in favor of TS version
      'no-unused-vars': 'off',

      // React rules
      ...react.configs.recommended.rules,
      'react/react-in-jsx-scope': 'off',
      'react/prop-types': 'off',

      // React Hooks rules
      ...reactHooks.configs.recommended.rules,
      // Disable overly strict rules
      'react-hooks/set-state-in-effect': 'off', // Valid pattern for dialog state reset
      'react-hooks/incompatible-library': 'warn', // React Hook Form pattern
      'react/display-name': 'off', // Not needed for test wrappers
    },
  },

  // Playwright test files - disable React hooks rules (Playwright's `use` is not a React hook)
  {
    files: ['tests/**/*.ts', 'tests/**/*.tsx'],
    rules: {
      'react-hooks/rules-of-hooks': 'off',
      'react-hooks/exhaustive-deps': 'off',
    },
  },

  // JavaScript files
  {
    files: ['**/*.js', '**/*.mjs'],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: 'module',
      globals: {
        console: 'readonly',
        process: 'readonly',
        __dirname: 'readonly',
        __filename: 'readonly',
        module: 'readonly',
        require: 'readonly',
        Buffer: 'readonly',
        global: 'readonly',
      },
    },
  },

  // Rule 21 enforcement (story 19-3) — every file under apps/web/src/components/
  // that renders designed UI MUST carry a leading `// Implements: Component/{Name}
  // ({penNodeId})` header (or a documented exemption). Scoped here, not in the rule.
  // Hooks/services/stores/utils/route files are out of scope by virtue of not
  // matching `components/**`; spec/test files and index.ts barrels are ignored.
  //
  // Each rule gets its own config block so the spec-side `flatConfig.filter` test
  // can assert "exactly one config block enables this rule". Both blocks share
  // identical scoping and reference the same `localRules` plugin object — flat
  // config dedup is a no-op when the value is the same identity.
  {
    files: ['apps/web/src/components/**/*.{ts,tsx}'],
    ignores: [
      'apps/web/src/components/**/*.spec.{ts,tsx}',
      'apps/web/src/components/**/*.test.{ts,tsx}',
      'apps/web/src/components/**/index.ts',
    ],
    plugins: {
      local: localRules,
    },
    rules: {
      'local/implements-pen-node-id': 'error',
    },
  },

  // Rule 23 enforcement (story 19-9) — every file under apps/web/src/components/
  // whose source reads the wall clock (Date.now / Date.UTC / Date.parse / new Date())
  // MUST carry a leading `// Clock-mocked:` / `// Clock-injected:` /
  // `// Time-bomb-exempt:` header. Same scoping as Rule 21 (above) so a
  // refactor of one block carries to the other; same `ignores` list for
  // spec/test/barrel files.
  {
    files: ['apps/web/src/components/**/*.{ts,tsx}'],
    ignores: [
      'apps/web/src/components/**/*.spec.{ts,tsx}',
      'apps/web/src/components/**/*.test.{ts,tsx}',
      'apps/web/src/components/**/index.ts',
    ],
    plugins: {
      local: localRules,
    },
    rules: {
      'local/time-dependent-fixture-stability': 'error',
    },
  },

  // Epic 11 Retro AI-1 — eslint-plugin-jsx-a11y enforcement. Scope mirrors the
  // Rule 21/23 component blocks (same files/ignores) so a scope refactor of one
  // carries intent to all three. WARN (not error) preserves the `lint:all`
  // 0-errors gate while the existing component a11y-violation batch surfaces as
  // warnings for retro-11-AI1b to clear; the warn→error ratchet is AI1b's
  // closing move, NOT this story.
  //
  // jsx-a11y's recommended ruleset ships at 'error'; remap every recommended
  // rule key to 'warn' rather than hand-listing rules, so the enabled set stays
  // current with the plugin version.
  {
    files: ['apps/web/src/components/**/*.{ts,tsx}'],
    ignores: [
      'apps/web/src/components/**/*.spec.{ts,tsx}',
      'apps/web/src/components/**/*.test.{ts,tsx}',
      'apps/web/src/components/**/index.ts',
    ],
    plugins: { 'jsx-a11y': jsxA11y },
    rules: Object.fromEntries(
      Object.keys(jsxA11y.flatConfigs.recommended.rules).map((r) => [r, 'warn'])
    ),
  },

  // Prettier config (must be last)
  prettier,
];
