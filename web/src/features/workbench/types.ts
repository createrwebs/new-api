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
export type WorkbenchMediaKind = 'image' | 'video'

/** A catalog model the current user can drive from the workbench. */
export interface WorkbenchModel {
  /** Model name sent upstream. */
  id: string
  /** Which relay endpoint serves this model. */
  kind: WorkbenchMediaKind
}

export interface GenerateImageInput {
  apiKey: string
  model: string
  prompt: string
  count: number
  size: string
}

export interface GeneratedMedia {
  /** Stable id for list keys; derived from the response index. */
  id: string
  url: string
  revisedPrompt?: string
}

export interface WorkbenchGenerateResult {
  created: number
  items: GeneratedMedia[]
}