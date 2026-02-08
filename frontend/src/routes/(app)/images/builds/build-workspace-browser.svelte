<script lang="ts">
	import { onMount } from 'svelte';
	import FileList from '$lib/components/file-browser/FileList.svelte';
	import FileBreadcrumb from '$lib/components/file-browser/FileBreadcrumb.svelte';
	import CreateFolderDialog from '$lib/components/file-browser/CreateFolderDialog.svelte';
	import FileUploadDialog from '$lib/components/file-browser/FileUploadDialog.svelte';
	import { ArcaneButton } from '$lib/components/arcane-button/index.js';
	import { UploadIcon, MoveToFolderIcon, EllipsisIcon, CopyIcon } from '$lib/icons';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import { toast } from 'svelte-sonner';
	import { UseClipboard } from '$lib/hooks/use-clipboard.svelte';
	import type { FileEntry } from '$lib/types/file-browser.type';
	import type { FileProvider } from '$lib/components/file-browser';

	let {
		provider,
		rootLabel = '/builds',
		rootPath,
		persistKey = 'arcane-build-workspace-table',
		onSelectContext
	}: {
		provider: FileProvider;
		rootLabel?: string;
		rootPath?: string;
		persistKey?: string;
		onSelectContext?: (path: string) => void;
	} = $props();

	let currentPath = $state('/');
	let files = $state<FileEntry[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let showCreateFolder = $state(false);
	let showUpload = $state(false);

	let editorOpen = $state(false);
	let editorFile = $state<FileEntry | null>(null);
	let editorContent = $state('');
	let editorLoading = $state(false);
	let editorSaving = $state(false);
	let editorError = $state<string | null>(null);

	const clipboard = new UseClipboard();

	const absoluteCurrentPath = $derived.by(() => {
		const root = (rootPath ?? '').trim();
		if (!root) return currentPath;
		const normalizedRoot = root.endsWith('/') ? root.slice(0, -1) : root;
		if (currentPath === '/' || currentPath === '') return normalizedRoot;
		return `${normalizedRoot}${currentPath.startsWith('/') ? '' : '/'}${currentPath}`;
	});

	function b64DecodeUnicode(str: string) {
		try {
			return decodeURIComponent(
				atob(str)
					.split('')
					.map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
					.join('')
			);
		} catch {
			return atob(str);
		}
	}

	async function loadFiles(path: string) {
		loading = true;
		error = null;
		try {
			const result = await provider.list(path);
			files = result.sort((a, b) => {
				if (a.isDirectory && !b.isDirectory) return -1;
				if (!a.isDirectory && b.isDirectory) return 1;
				return a.name.localeCompare(b.name);
			});
			currentPath = path;
			onSelectContext?.(path);
		} catch (e: any) {
			error = e.message || 'Failed to load files';
		} finally {
			loading = false;
		}
	}

	function handleNavigate(path: string) {
		loadFiles(path);
	}

	async function handleCopyPath() {
		const status = await clipboard.copy(absoluteCurrentPath);
		if (status === 'success') {
			toast.success('Copied current path');
			return;
		}
		toast.error('Failed to copy path');
	}

	async function openEditor(file: FileEntry) {
		if (file.isDirectory) return;
		editorFile = file;
		editorOpen = true;
		editorLoading = true;
		editorError = null;
		try {
			const res = await provider.getContent(file.path);
			editorContent = b64DecodeUnicode(res.content);
		} catch (e: any) {
			editorError = e.message || 'Failed to load file';
			editorContent = '';
		} finally {
			editorLoading = false;
		}
	}

	async function handleSaveFile() {
		if (!editorFile) return;
		editorSaving = true;
		try {
			const dirPath = editorFile.path.split('/').slice(0, -1).join('/') || '/';
			const file = new File([editorContent], editorFile.name, { type: 'text/plain' });
			await provider.upload(dirPath, file);
			toast.success('File saved');
			editorOpen = false;
			await loadFiles(currentPath);
		} catch (e: any) {
			toast.error(e.message || 'Failed to save file');
		} finally {
			editorSaving = false;
		}
	}

	onMount(() => {
		loadFiles('/');
	});
</script>

<div class="flex h-full min-h-0 flex-col gap-5">
	<div class="border-border/40 flex flex-wrap items-start justify-between gap-4 border-b pt-1 pb-4">
		<div class="min-w-0 flex-1 space-y-3">
			<FileBreadcrumb path={currentPath} {rootLabel} onNavigate={handleNavigate} />
			<div class="flex min-w-0 items-center gap-3">
				<div class="text-muted-foreground min-w-0 flex-1 truncate font-mono text-[11px]" title={absoluteCurrentPath}>
					{absoluteCurrentPath}
				</div>
			</div>
		</div>

		<div class="flex shrink-0 flex-wrap items-center justify-end gap-3">
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<ArcaneButton
							{...props}
							action="base"
							tone="outline"
							size="icon"
							class="size-9"
							icon={EllipsisIcon}
							customLabel={m.common_actions()}
						/>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end" class="min-w-[180px]">
					<DropdownMenu.Item onclick={handleCopyPath}>
						<CopyIcon class="size-4" />
						Copy current path
					</DropdownMenu.Item>
					<DropdownMenu.Separator />
					<DropdownMenu.Item onclick={() => (showCreateFolder = true)}>
						<MoveToFolderIcon class="size-4" />
						{m.volumes_browser_new_folder()}
					</DropdownMenu.Item>
					<DropdownMenu.Item onclick={() => (showUpload = true)}>
						<UploadIcon class="size-4" />
						{m.volumes_browser_upload_files()}
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>
	</div>

	{#if loading}
		<div class="flex flex-1 items-center justify-center p-8">
			<Spinner class="text-muted-foreground size-8" />
		</div>
	{:else if error}
		<div class="border-destructive/20 bg-destructive/10 text-destructive rounded-lg border p-6 text-sm">
			{error}
		</div>
	{:else}
		<div class="min-h-0 flex-1 overflow-hidden">
			<FileList
				{files}
				{currentPath}
				{persistKey}
				minimal
				onNavigate={handleNavigate}
				onRefresh={() => loadFiles(currentPath)}
				onDelete={(file) => provider.delete(file.path)}
				onDownload={(file) => provider.download(file.path)}
				onPreview={openEditor}
			/>
		</div>
	{/if}
</div>

<CreateFolderDialog
	bind:open={showCreateFolder}
	{currentPath}
	onCreate={async (name) => {
		const fullPath = currentPath === '/' ? `/${name}` : `${currentPath}/${name}`;
		await provider.mkdir(fullPath);
		await loadFiles(currentPath);
	}}
/>

<FileUploadDialog
	bind:open={showUpload}
	{currentPath}
	onUpload={async (file) => {
		await provider.upload(currentPath, file);
		await loadFiles(currentPath);
	}}
/>

<Dialog.Root bind:open={editorOpen}>
	<Dialog.Content class="max-w-3xl">
		<Dialog.Header>
			<Dialog.Title>{editorFile?.name ?? 'File'}</Dialog.Title>
			<Dialog.Description class="break-all">{editorFile?.path}</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2">
			{#if editorLoading}
				<div class="flex items-center justify-center py-8">
					<Spinner class="text-muted-foreground size-8" />
				</div>
			{:else if editorError}
				<div class="border-destructive/20 bg-destructive/10 text-destructive rounded-lg border p-4 text-sm">
					{editorError}
				</div>
			{:else}
				<div class="space-y-2">
					<Label>File contents</Label>
					<Textarea rows={18} bind:value={editorContent} class="font-mono text-xs" />
					<p class="text-muted-foreground text-xs">Saving will overwrite the file contents.</p>
				</div>
			{/if}
		</div>
		<Dialog.Footer class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<ArcaneButton action="cancel" tone="outline" type="button" onclick={() => (editorOpen = false)} />
			<ArcaneButton
				action="save"
				type="button"
				onclick={handleSaveFile}
				disabled={editorLoading || editorSaving || !!editorError}
				loading={editorSaving}
			/>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
