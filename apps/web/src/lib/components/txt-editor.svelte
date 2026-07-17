<script lang="ts">
	// import { onMount, onDestroy } from 'svelte';
	// import { Editor } from '@tiptap/core';
	// import Document from '@tiptap/extension-document';
	// import Paragraph from '@tiptap/extension-paragraph';
	// import Text from '@tiptap/extension-text';
	// import { Button } from './ui/button';
	// let element = $state<
	// 	Element | { mount: HTMLElement } | ((editor: HTMLElement) => void) | null | undefined
	// >();
	// let editorState = $state<{ editor: Editor | null }>({ editor: null });
	// onMount(() => {
	// 	editorState.editor = new Editor({
	// 		element: element,
	// 		extensions: [Document, Paragraph, Text],
	// 		content: ``,
	// 		onTransaction: ({ editor }) => {
	// 			editorState = { editor };
	// 		}
	// 	});
	// });
	// onDestroy(() => {
	// 	editorState.editor?.destroy();
	// });
	// function printLines() {
	// 	if (!editorState.editor) return;
	// 	const text = editorState.editor.getText();
	// 	const lines = text.split('\n');
	// 	console.log(lines);
	// }
</script>

<div style="position: relative" class="app border">
	<!-- <Button onclick={printLines}>Print Lines</Button>
	<div bind:this={element} class="border-none outline-none min-h-[100px] p-2"></div> -->
</div>
