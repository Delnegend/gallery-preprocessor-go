<!-- eslint-disable vue/multi-word-component-names -->
<script setup lang="ts">

import { Primitive, type PrimitiveProps } from "reka-ui";

import { cn } from "../../lib/utils";

interface Props extends PrimitiveProps {
	variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
	size?: "default" | "xs" | "sm" | "lg" | "icon";
	class?: string;
}

const props = withDefaults(defineProps<Props>(), {
	as: "button",
	variant: "default",
	size: "default",
});
</script>

<template>
	<Primitive
		:as="as"
		:as-child="asChild"
		:class="cn(
			'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
			{
				'bg-primary text-primary-foreground shadow hover:bg-primary/90': props.variant === 'default',
				'bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90': props.variant === 'destructive',
				'border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground': props.variant === 'outline',
				'bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80': props.variant === 'secondary',
				'hover:bg-accent hover:text-accent-foreground': props.variant === 'ghost',
				'text-primary underline-offset-4 hover:underline': props.variant === 'link',
			},
			{
				'h-9 px-4 py-2': props.size === 'default',
				'h-7 rounded px-2': props.size === 'xs',
				'h-8 rounded-md px-3 text-xs': props.size === 'sm',
				'h-10 rounded-md px-8': props.size === 'lg',
				'h-9 w-9': props.size === 'icon',
			},
			props.class)">
		<slot />
	</Primitive>
</template>
