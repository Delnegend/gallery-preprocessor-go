<script setup lang="tsx">
import { onMounted, onUnmounted, ref } from "vue";

import { backend, main } from "../wailsjs/go/models";
import { EventsEmit, EventsOn, OnFileDrop, OnFileDropOff } from "../wailsjs/runtime/runtime";
import Button from "./components/ui/Button.vue";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "./components/ui/hover-card";
import { cn } from "./lib/utils";

const progress = ref<number>(0.0);
const cmdOutputs = ref<string[]>([]);
EventsOn(main.OtherEmitID.Progress, (data: number) => { progress.value = data; });
EventsOn(main.OtherEmitID.Warning, (data: string) => cmdOutputs.value.push(data));

const tasks = ([
	[
		backend.TaskID.Artefact,
		"Artefact",
		"Remove JPEG artifacts and output PNG.<br /><br />Accepts: JPG"
	],
	[
		backend.TaskID.ArtefactJxl,
		"Artefact + CJXL (Lossy)",
		"Remove JPEG artifacts and output PNG, then compress to JXL (lossy).<br /><br />Accepts: <code>.jpg</code>"
	],
	[
		backend.TaskID.CjxlLossless,
		"CJXL (Lossless)",
		"Compress JPG/PNG to JXL (lossless).<br /><br />Accepts: <code>.jpg</code>, <code>.png</code>"
	],
	[
		backend.TaskID.CjxlLossy,
		"CJXL (Lossy)",
		"Compress JPG/PNG to JXL (lossy).<br /><br />Accepts: <code>.jpg</code>, <code>.png</code>"

	],
	[
		backend.TaskID.Djxl,
		"DJXL",
		"Decompress JXL to JPG/PNG.<br /><br />Accepts: <code>.jxl</code>"
	],
	[
		backend.TaskID.Par2,
		"PAR2",
		"Create parity files for 7z.<br /><br />Accepts: <code>.7z</code>"
	],
] satisfies Array<[backend.TaskID, string, string]>)
	.map(([ID, Label, Description]) => ({
		ID,
		Label,
		Description,
		Bounds: { X: 0, Y: 0, Width: 0, Height: 0 }
	}));

const resizeObserver = new ResizeObserver((entries) => {
	for (const entry of entries) {
		const { target } = entry;
		const task = tasks.find(task => task.ID === target.id as backend.TaskID);
		if (task) {
			const { width, height, x, y } = target.getBoundingClientRect();
			task.Bounds = { X: x, Y: y, Width: width, Height: height };
		}
	}
});

onMounted(() => {
	document.querySelector("html")?.classList.add("dark");

	for (const task of tasks
		.map(task => document.getElementById(task.ID))
		.filter(task => task !== null)
	) {
		resizeObserver.observe(task);
	}
});

// this exists just to takeover webview's drag and drop event
OnFileDrop(() => { /** */ }, false);
EventsOn("wails:file-drop", (x, y, paths) => {
	const droppedOn = tasks.find((task) => {
		const { X, Y, Width, Height } = task.Bounds;
		return x >= X && x <= X + Width && y >= Y && y <= Y + Height;
	});
	if (droppedOn) {
		cmdOutputs.value = [];
		EventsEmit(droppedOn.ID, paths);
	}
});

const runningTask = ref<backend.TaskID | null>(null);
EventsOn(main.OtherEmitID.TaskStart, (taskID: backend.TaskID) => {
	runningTask.value = taskID;
});
EventsOn(main.OtherEmitID.TaskDone, () => {
	runningTask.value = null;
	progress.value = 0.0;
});

onUnmounted(() => {
	OnFileDropOff();
	resizeObserver.disconnect();
});
</script>

<template>
	<div class="grid grid-rows-[auto,auto,1fr] h-dvh">
		<Button
			variant="destructive"
			class="h-12 w-full rounded-none text-xl top-0"
			size="lg"
			:onclick="() => EventsEmit(main.OtherEmitID.CancelTask, runningTask)"
			:disabled="runningTask === null">
			Cancel Task
		</Button>

		<div
			class="relative grid grid-cols-3 grid-rows-2">
			<div
				class="absolute left-0 top-0 -z-10 size-full bg-secondary transition-transform"
				:style="{
					transform: `translateX(-${(1 - progress) * 100}%)`,
				}" />
			<div
				v-for="task in tasks"
				:key="task.ID"
				:id="task.ID"
				:class="cn(
					'border text-base border-secondary-foreground/30 px-3 py-2 flex justify-center items-center h-full text-center',
					runningTask === task.ID && 'animate-bounce',
					runningTask !== null && runningTask !== task.ID && 'text-secondary-foreground/50'
				)">
				<HoverCard :openDelay="100" :closeDelay="100">
					<HoverCardTrigger class="hover:underline underline-offset-4">
						{{ task.Label }}
					</HoverCardTrigger>
					<HoverCardContent class="text-balance text-sm px-3 py-2 text-primary/80">
						<span v-html="task.Description" />
					</HoverCardContent>
				</HoverCard>
			</div>
		</div>

		<div class="overflow-y-auto flex flex-col gap-2">
			<code v-for="[index, output] in cmdOutputs.entries()" :key="index">
				{{ output }}
			</code>
		</div>
	</div>
</template>
