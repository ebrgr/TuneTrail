// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

import tailwindcss from "@tailwindcss/vite";

// https://astro.build/config
export default defineConfig({
  integrations: [
    starlight({
      head: [
        {
          tag: "script",
          attrs: {
            src: "https://static.cloudflareinsights.com/beacon.min.js",
            "data-cf-beacon": '{"token": "1948caaaba10463fa1d310ee02b0951c"}',
            defer: true,
          },
        },
      ],
      title: "TuneTrail",
      logo: {
        src: "./src/assets/tunetrail.png",
        replacesTitle: false,
      },
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/ebrgr/TuneTrail",
        },
      ],
      sidebar: [
        {
          label: "Guides",
          items: [
            // Each item here is one entry in the navigation menu.
            { label: "Installation", slug: "guides/installation" },
            { label: "Listening profiles", slug: "guides/profiles" },
            { label: "Importing Data", slug: "guides/importing" },
            { label: "Active ingestion pilot", slug: "guides/active-ingestion" },
            { label: "Setting up the Scrobbler", slug: "guides/scrobbler" },
            { label: "Editing Data", slug: "guides/editing" },
          ],
        },
        {
          label: "Quickstart",
          items: [
            { label: "Setup with Navidrome", slug: "quickstart/navidrome" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "Configuration Options", slug: "reference/configuration" },
          ],
        },
      ],
      customCss: [
        // Path to your Tailwind base styles:
        "./src/styles/global.css",
      ],
    }),
  ],

  site: "https://koito.io",

  vite: {
    plugins: [tailwindcss()],
  },
});
