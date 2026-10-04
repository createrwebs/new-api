/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
/**
 * Available font names (visit the url `/settings/appearance`).
 *
 * The project ships a single family — IBM Plex Sans Thai — so this list has one
 * entry and `FontProvider` therefore always resolves to it. The switcher UI is
 * kept (rather than deleted) so the surrounding appearance controls stay
 * intact if more families are added later.
 *
 * 📝 How to Add a New Font (Tailwind v4+):
 * 1. Add the font name here.
 * 2. Load the family: a `<link>` tag in 'index.html' (CDN, e.g. Google Fonts)
 *    or an `@import` of a self-hosted package in 'src/styles/index.css'.
 * 3. Add the font family to 'theme.css' using the `@theme inline`
 *    `font-family` CSS variable.
 *
 * Example:
 * fonts.ts           → Add 'roboto' to this array.
 * index.html         → Add font link for Roboto.
 * theme.css          → Add the new font in the CSS, e.g.:
 *   @theme inline {
 *      // ... other font families
 *      --font-roboto: 'Roboto', var(--font-sans);
 *   }
 */
export const fonts = ['ibm-plex-sans-thai'] as const
