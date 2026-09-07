import { createContext } from "react";

export const LabelColorsContext = createContext<Record<string, string>>({});
export const HideDefaultLabelsContext = createContext<boolean>(false);
export const DefaultLabelsContext = createContext<{ name: string; color: string }[]>([]);
