import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'doc',
      id: 'README',
      label: 'Introduction',
    },
    {
      type: 'doc',
      id: 'getting-started',
      label: 'Getting Started',
    },
    {
      type: 'doc',
      id: 'cli',
      label: 'CLI Reference',
    },
    {
      type: 'category',
      label: 'Core Concepts',
      items: [
        'architecture',
        'run-modes',
        'configuration',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        'custom-routes',
        'testing',
        'best-practices',
      ],
    },
  ],
};

export default sidebars;
