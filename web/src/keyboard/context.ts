import { createContext } from "react";
import type { KeyboardRegistry } from "./registry";

export const KeyboardNavContext = createContext<KeyboardRegistry | null>(null);
