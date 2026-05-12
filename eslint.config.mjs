import js from '@eslint/js';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import prettier from 'eslint-config-prettier';
// Local custom rules (story 19-3) — CJS plugin, default-imported as its module.exports.
import localRules from './apps/web/src/eslint-rules/implements-pen-node-id.js';

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

  // Prettier config (must be last)
  prettier,
];
