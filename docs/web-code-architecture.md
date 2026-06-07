# Winterfell Frontend Architecture Context

This document captures the core mindset, module design, boundaries, and implementation philosophy visible in the Winterfell frontend. It intentionally avoids tying the architecture to one framework. The same patterns can be carried into React, Svelte, Vue, TanStack Router, Solid, or another UI stack.

## Core Mindset

Winterfell is not structured like a generic dashboard. It is a product-specific contract workbench for generating, inspecting, building, and exporting Solana Anchor programs.

The frontend architecture is driven by product language first:

- A user does not manage records; they build contracts.
- A screen is not a generic page; it is a product surface such as landing, marketplace, docs, or builder playground.
- State is not global by default; it belongs to a domain such as builder chat, code editor, terminal, templates, docs, or user session.
- Components are not organized around technology primitives first; they are organized around visible product concepts.

The best code in this frontend tends to be:

- Explicit rather than clever.
- Domain-named rather than generic.
- Small at module boundaries, direct inside implementations.
- Stable at interfaces, flexible inside components/stores/services.
- Optimistic in UI flow, defensive at API boundaries.

## Product Surfaces

The frontend has four conceptual surfaces:

- Marketing: explains the product and converts visitors.
- Marketplace/home: shows templates, user contracts, and recent builds.
- Builder playground: chat, editor, files, plans, terminal, export actions.
- Docs: explains workflows through structured client-rendered content.

Each surface gets its own component area. This matters more than framework route names. In any framework, preserve this mental split:

```txt
frontend/
  surfaces/
    marketing/
    home/
    builder/
    docs/
    pricing/
  shared/
    ui/
    hooks/
    lib/
    providers/
  state/
    user/
    code/
    builder/
    docs/
  services/
    marketplace/
    generation/
    github/
    editor/
```

Current code uses this shape through `apps/web/src/components/*`, `src/store/*`, `src/hooks/*`, and `src/lib/server/*`.

## Module Structure Philosophy

Modules are grouped by domain responsibility, not by technical file type alone.

The main module kinds are:

- Surface modules: compose a complete screen or workflow region.
- Product components: implement visible product concepts.
- State modules: own persistent UI/domain state and mutations.
- Service modules: own backend calls and response fallbacks.
- Hook/behavior modules: own reusable workflow or DOM behavior.
- Utility modules: pure helpers with no product lifecycle.
- Type modules: shared contracts between frontend/backend or app-local UI types.

The rule is: put code where its language belongs.

Examples:

- `BuilderDashboard` belongs in builder components because it composes the builder workbench.
- `useBuilderChatStore` belongs in code/builder state because chat is contract-scoped builder state.
- `Marketplace.getTemplates` belongs in a service because endpoint construction and fallback behavior are not component concerns.
- `cn` belongs in utilities because it is framework-adjacent and domain-neutral.

## Interface And Implementation Pattern

Good modules expose a small interface and hide implementation details.

The interface is what other modules are allowed to know. The implementation is how the module fulfills that contract.

### State Module Interface

The builder chat state exposes explicit operations:

```ts
interface BuilderChatState {
    contracts: Record<string, ContractState>;
    currentContractId: string | null;
    setCurrentContractId: (contractId: string) => void;
    getCurrentContract: () => ContractState;
    setLoading: (loading: boolean) => void;
    setMessage: (message: Message) => void;
    upsertMessage: (message: Partial<Message> & { id: string }) => void;
    cleanContract: (contractId: string) => void;
}
```

This interface says what the rest of the app can do:

- Select the current contract.
- Read current contract state.
- Append or update messages.
- Toggle loading.
- Clean up a contract session.

The implementation detail is that contracts are stored as `Record<string, ContractState>`. Most components should not care how updates are stored internally; they should call the provided operations.

Why this is good:

- It prevents random modules from mutating nested state however they want.
- It makes lifecycle operations obvious.
- It allows later internal changes without rewriting every component.

Portable version:

```ts
type BuilderChatPort = {
    selectContract(contractId: string): void;
    current(): ContractState;
    appendMessage(message: Message): void;
    upsertMessage(message: Partial<Message> & { id: string }): void;
    setLoading(value: boolean): void;
    dispose(contractId: string): void;
};
```

This can be implemented with Zustand, Svelte stores, Vue refs, TanStack Store, Redux, signals, or a class. The architecture is the port, not the library.

### Service Module Interface

Service modules expose use-case methods, not HTTP details:

```ts
export default class Marketplace {
    public static async getUserContracts(token: string): Promise<Contract[]> {
        try {
            const { data } = await axios.get(GET_USER_CONTRACTS, {
                headers: { Authorization: `Bearer ${token}` },
            });

            return data.data;
        } catch (error) {
            console.error('Failed to fetch user contracts', error);
            return [];
        }
    }
}
```

The public interface is `getUserContracts(token): Promise<Contract[]>`.

The implementation details are:

- Axios is used.
- URL comes from centralized constants.
- Authorization header is attached.
- Failure returns `[]`.

Components should not know endpoint paths, raw response envelopes, or fallback details.

Portable version:

```ts
interface MarketplaceService {
    getUserContracts(token: string): Promise<Contract[]>;
    getAllContracts(token: string): Promise<Contract[]>;
    getTemplates(): Promise<Template[]>;
}
```

This interface can be implemented using Axios, fetch, GraphQL, RPC, server functions, or TanStack Query query functions.

### Component Module Interface

Components expose props/events and hide internal UI state.

Example mindset:

```tsx
<BuilderDashboard />
```

The parent does not pass every chat message, terminal log, side panel, file tree, and loading flag. The dashboard owns the builder composition and reads from domain stores. This keeps high-level surfaces clean.

When a component needs a public interface, prefer product-language props:

```ts
type ContractReviewCardProps = {
    contractId: string;
    open: boolean;
    onClose: () => void;
    onSubmit: () => void;
};
```

Avoid generic props like `data`, `config`, or `options` unless the abstraction is genuinely reusable.

## Boundary Pattern

The frontend has clear boundaries between layers.

```txt
Surface composition
  -> Product components
    -> State modules / hooks
      -> Service modules
        -> Backend endpoints
```

Data should generally flow down through composition and state selectors. Actions should flow through explicit event handlers or state/service operations.

Avoid these boundary leaks:

- Components constructing raw backend URLs.
- Deep child components receiving huge prop bags from route/surface files.
- UI components mutating service response shapes directly.
- Service modules importing UI components or stores.
- Generic shared modules knowing about builder-specific concepts.

## Contract-Scoped State Pattern

The builder playground is keyed by `contractId`. That shape drives one of the most important state decisions: builder chat state is contract-scoped.

```ts
type BuilderState = {
    contracts: Record<string, ContractState>;
    currentContractId: string | null;
};
```

Why:

- The URL or active workspace identifies the contract.
- Chat, loading state, selected template, and current editing context belong to a contract.
- Moving between contracts should not leak messages or generated state.

How:

```ts
function setCurrentContractId(contractId: string) {
    state.currentContractId = contractId;

    if (!state.contracts[contractId]) {
        state.contracts[contractId] = getDefaultContractState();
    }
}
```

Cleanup is part of the lifecycle:

```ts
function disposeContract(contractId: string) {
    delete state.contracts[contractId];
    resetEditor();
    resetSocket();
}
```

Portable rule:

- If state belongs to a contract, key it by `contractId`.
- If state belongs to the whole UI, put it in a UI/domain store.
- If state belongs only to one panel, keep it local.

## Optimistic Workflow Pattern

Generation is designed to feel immediate. The frontend initializes local state before the backend finishes work.

```ts
function startGeneration(contractId: string, instruction: string, templateId?: string) {
    builderChat.selectContract(contractId);
    builderChat.appendMessage({
        id: uuid(),
        contractId,
        role: ChatRole.USER,
        content: instruction,
        stage: STAGE.START,
        createdAt: new Date(),
    });

    navigateToBuilder(contractId);
    generationService.start(contractId, instruction, templateId);
}
```

Why:

- The user gets instant feedback.
- The active workspace exists before generated files arrive.
- Backend generation can stream or complete asynchronously.

Portable rule:

- Put user intent into local state first.
- Start the backend operation second.
- Let streaming/polling/service responses reconcile state later.

## Builder Workbench Pattern

The builder is a workbench, not a form. It has multiple synchronized regions:

- Chat: intent, messages, templates, generation controls.
- Side rail: switches between files, GitHub, plan, or other builder panels.
- File tree: navigates generated source files.
- Editor: displays selected generated code.
- Plan panel: shows planned/executed generation steps.
- Terminal: shows command/build logs.

The workbench composition follows a stable shell:

```tsx
function BuilderDashboard() {
    return (
        <BuilderShell>
            <BuilderChats />
            <EditorSidePanel />
            <SidePanel>{renderSidePanel()}</SidePanel>
            {renderMainPanel()}
            <Terminal />
        </BuilderShell>
    );
}
```

Panel selection is explicit:

```ts
function renderMainPanel(currentState: SidePanelValue) {
    switch (currentState) {
        case 'file':
            return CodeEditor;
        case 'github':
            return CodeEditor;
        case 'plan':
            return PlanPanel;
    }
}
```

Why:

- The shell remains understandable.
- New panels have one obvious integration point.
- Product concepts stay visible in code.

## File Model Pattern

Generated files arrive as flat paths and content:

```ts
type FileContent = {
    path: string;
    content: string;
};
```

The UI needs a navigable tree:

```ts
type FileNode = {
    id: string;
    name: string;
    type: 'file' | 'folder';
    content?: string;
    children?: FileNode[];
};
```

The conversion belongs in the editor state/module, not inside the file tree component:

```ts
function parseFileStructure(files: FileContent[]): FileNode {
    for (const { path, content } of files) {
        const parts = path.split('/').filter(Boolean);
        const fileName = parts.pop();
        const parentFolder = findOrCreateFolder(root, parts);

        parentFolder.children.push({
            id: path,
            name: fileName,
            type: 'file',
            content,
        });
    }

    return root;
}
```

Why:

- Backend payload stays simple.
- UI gets the structure it needs.
- File tree, editor, and sync behavior share one canonical file model.

## Naming Pattern

Names should reveal product intent.

Good names in this codebase:

- `BuilderDashboard`
- `BuilderChats`
- `BuilderTemplatesPanel`
- `CodeEditor`
- `Filetree`
- `PlanPanel`
- `Terminal`
- `ContractTemplates`
- `MostRecentBuilds`
- `Marketplace`
- `GenerateContract`

Bad direction:

- `DataPanel`
- `MainContent`
- `Widget`
- `ApiUtil`
- `Manager`
- `Container`

Use names from the user's mental model, not from the framework's vocabulary.

## State Placement Rules

Use this decision tree:

- If many modules need it and it has product meaning, put it in a domain state module.
- If it is tied to a contract, key it by `contractId`.
- If it controls only one component's open/closed/input state, keep it local.
- If it comes from the backend and needs caching/loading/error semantics, put it behind a service/query boundary.
- If it is derived from other state, compute it near the consumer unless it is expensive or reused.

Examples:

- `currentContractId`: domain state.
- `messages` per contract: contract-scoped domain state.
- `showTemplatePanel`: local component state.
- `templates`: user/home/builder shared domain state.
- `currentCode` and `currentFile`: editor domain state.
- `isCommandRunning`: terminal domain state.

## Service And Endpoint Rules

Services represent use cases. Endpoints are implementation details.

Endpoint constants are centralized:

```ts
export const API_URL = BACKEND_URL + '/api/v1';
export const GET_USER_CONTRACTS = API_URL + '/contracts/get-user-contracts';
export const GET_ALL_TEMPLATES = API_URL + '/template/get-templates';
```

Services expose product operations:

```ts
class Marketplace {
    static getUserContracts(token: string): Promise<Contract[]>;
    static getAllContracts(token: string): Promise<Contract[]>;
    static getTemplates(): Promise<Template[]>;
}
```

Rules:

- Components do not build endpoint strings.
- Components do not parse response envelopes when a service can return domain data.
- Services return stable fallbacks like `[]`, `false`, or `null` when that is the current app convention.
- If the user must act on failure, pair the fallback with a visible error/toast at the UI boundary.

## Hook Or Behavior Module Pattern

Hooks in the current code are behavior modules. In another framework, these could be composables, actions, stores, or plain functions.

Examples:

- `useGenerate`: generation workflow.
- `useWebSocket`: socket lifecycle and message publishing.
- `useCurrentContract`: selector for active contract state.
- `useHandleClickOutside`: DOM behavior.
- `useResize`, `useTerminalResize`: layout behavior.
- `useShortcut`: keyboard behavior.

Portable pattern:

```ts
function createGenerateWorkflow(deps: {
    chat: BuilderChatPort;
    generation: GenerationService;
    navigate: (contractId: string) => void;
    session: SessionReader;
}) {
    return {
        setInitialState,
        startGeneration,
    };
}
```

This is the same architecture whether implemented as a React hook, Svelte module, Vue composable, or plain TypeScript function.

## UI Design Philosophy

The UI identity is dark, terminal-like, editor-like, and command-driven.

Common visual language:

- Dark surfaces: `darkest`, `dark`, neutral black panels.
- Soft borders: neutral or light alpha borders.
- Low-radius controls: sharp, tool-like surfaces.
- Muted text with selective bright emphasis.
- Motion that supports layout transitions, not decorative noise.
- Marketing can be cinematic; builder surfaces should stay focused and workbench-like.

Design rule:

- Do not introduce generic SaaS cards if the surface is a contract workbench.
- Do not over-abstract styling into theme systems unless repetition proves the need.
- Preserve visible product identity even if the framework changes.

## Events And Lifecycle Pattern

Some behaviors cross component boundaries through lifecycle hooks or browser events.

Example: editor keyboard command opens global search:

```ts
editor.addCommand(CtrlOrCmdP, () => {
    window.dispatchEvent(new CustomEvent('open-search-bar'));
});
```

This is acceptable for cross-cutting UI behavior when direct parent-child wiring would be awkward.

Lifecycle cleanup is equally important:

```ts
function disposeBuilderWorkspace(contractId: string) {
    cleanContract(contractId);
    resetContractId();
    resetEditor();
    cleanWebSocketClient();
}
```

Rule:

- If a module creates a lifecycle resource, it must own or clearly participate in cleanup.
- Sockets, timers, subscriptions, and route/workspace state should not survive accidental context changes.

## Error Handling Philosophy

The current code usually fails soft at service boundaries:

```ts
try {
    return await fetchTemplates();
} catch (error) {
    console.error('Failed to fetch templates', error);
    return [];
}
```

Why:

- UI modules stay simple.
- A missing optional section does not crash the whole workbench.
- Failure shape is predictable.

Tradeoff:

- Silent fallback can hide important failures.
- For user-triggered actions, add visible feedback.

Rule:

- Background/non-critical fetch: safe fallback is fine.
- User action: show an error state or toast.
- Data needed to continue: represent loading/error explicitly.

## What To Keep Portable Across Frameworks

These are the actual architectural ideas worth reusing:

- Domain-first module names.
- Product-surface folder boundaries.
- Small public interfaces around stores/services/workflows.
- Contract-scoped state keyed by `contractId`.
- Optimistic workflow initialization before backend completion.
- Centralized endpoint/service boundary.
- Stable workbench shell with explicit panel switching.
- Backend file payload converted into UI file model at the editor-state boundary.
- Local state for purely local UI concerns.
- Explicit lifecycle cleanup for sockets, stores, timers, and workspace state.

These are implementation details, not architectural requirements:

- The specific routing library.
- The specific component framework.
- Zustand as the store library.
- Axios as the HTTP client.
- Monaco as the editor.
- Tailwind as the styling engine.

## Current Caveats

These are visible in the current code and should not be copied blindly:

- Some naming has typos or inconsistent casing, such as `SessionSeter`, `setCollapsechat`, and `ExportPanel..tsx`.
- Some modules named `server` are imported from client-side code; conceptually they are API services.
- Some effects suppress dependency linting.
- WebSocket initialization is currently commented out in `useWebSocket`.
- `useCodeEditor.syncFiles` contains a placeholder token.
- Some styles are bespoke arbitrary values instead of reusable tokens.

Treat these as local historical facts, not desired architecture.

## How To Add A New Feature In This Style

1. Name the feature in product language.
2. Identify the surface it belongs to: marketing, home, builder, docs, pricing.
3. Define the smallest public interface for its state/service/workflow.
4. Keep implementation details inside the owning module.
5. Add shared state only if multiple modules need it.
6. If state is contract-specific, key it by `contractId`.
7. Put backend calls behind a service operation.
8. Keep the surface composition readable.
9. Preserve the visual identity of the surrounding surface.
10. Add cleanup for any lifecycle resource.

Example module plan:

```txt
builder/
  components/
    DeploymentPanel.tsx
  state/
    deploymentState.ts
  services/
    deploymentService.ts
  workflows/
    deploymentWorkflow.ts
```

Example public contracts:

```ts
interface DeploymentService {
    startDeployment(token: string, contractId: string): Promise<DeploymentResult>;
    getDeploymentStatus(token: string, deploymentId: string): Promise<DeploymentStatus>;
}

interface DeploymentStatePort {
    setActiveDeployment(contractId: string, deployment: Deployment): void;
    appendLog(contractId: string, log: DeploymentLog): void;
    resetDeployment(contractId: string): void;
}
```

The framework can change. The boundary should remain stable.

## Good Code Definition

Good Winterfell-style frontend code is:

- Product-language first.
- Modular by domain.
- Interface-conscious.
- Explicit in state transitions.
- Defensive at backend boundaries.
- Optimistic where user flow benefits.
- Careful with lifecycle cleanup.
- Visually consistent with the command/workbench identity.

Bad Winterfell-style frontend code is:

- Generic dashboard structure with product names sprinkled on top.
- Direct endpoint calls from components.
- Huge prop chains where a domain store or workflow would be clearer.
- Shared abstractions created before repetition exists.
- Framework-specific patterns documented as if they are the architecture.
- Cleanup ignored for sockets, timers, subscriptions, or contract sessions.
