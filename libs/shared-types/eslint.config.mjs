import rootConfig from '../../eslint.config.mjs';
import nxEslintPlugin from '@nx/eslint-plugin';
import jsoncParser from 'jsonc-eslint-parser';

export default [
  ...rootConfig,
  {
    files: ['**/*.json'],
    plugins: {
      '@nx': nxEslintPlugin,
    },
    languageOptions: {
      parser: jsoncParser,
    },
    rules: {
      '@nx/dependency-checks': [
        'error',
        {
          ignoredFiles: ['{projectRoot}/eslint.config.{js,cjs,mjs,ts,cts,mts}'],
        },
      ],
    },
  },
];
