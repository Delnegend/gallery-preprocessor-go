import type { Updater } from "@tanstack/vue-table";
import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import type { Ref } from "vue";

export function cn(...inputs: ClassValue[]): string {
	return twMerge(clsx(inputs));
}

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-parameters, @typescript-eslint/no-explicit-any
export function valueUpdater<T extends Updater<any>>(updaterOrValue: T, ref: Ref): void {
	// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
	ref.value = typeof updaterOrValue === "function"
		// eslint-disable-next-line @typescript-eslint/no-unsafe-call
		? updaterOrValue(ref.value)
		: updaterOrValue;
}
