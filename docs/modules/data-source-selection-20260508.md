# Plan: Data Source Selection in Chat

## Overview
Add a data source selection feature to the chat creation page, allowing users to select Query Data Sources and pass them as parameters to the session creation flow.

## Requirement Doc
`docs/requirements/data-source-selection-in-chat.md`

## Implementation Steps

### Step 1: Update Settings Store
**File:** `frontend/src/stores/settings.ts`

Add `selectedDataSources` state and actions:
```typescript
// Add to settings state
selectedDataSources: [] as string[],

// Add actions
addDataSource(id: string) {
  if (!this.settings.selectedDataSources.includes(id)) {
    this.settings.selectedDataSources.push(id);
  }
},
removeDataSource(id: string) {
  this.settings.selectedDataSources = this.settings.selectedDataSources.filter(i => i !== id);
},
selectDataSources(ids: string[]) {
  this.settings.selectedDataSources = ids;
}
```

### Step 2: Create DataSourceSelector Component
**File:** `frontend/src/components/DataSourceSelector.vue`

Create a modal component that:
- Fetches query data sources via `listQueryDataSources()`
- Displays data sources in a list with checkbox selection
- Shows data source name, type (mysql/postgresql/sqlite), and status
- Includes "Add New Data Source" button that navigates to datasource management
- Emits selected data source IDs on confirm

### Step 3: Update Input-field Component
**File:** `frontend/src/components/Input-field.vue`

Add "+" button and dropdown menu:
1. Add a "+" button to the control-left section
2. Create a dropdown menu with options:
   - Upload File
   - Select Data Source
   - Select Knowledge Base
3. Add click handlers for each menu item
4. Add DataSourceSelector modal integration
5. Add data source chips display (similar to knowledge base chips)

### Step 4: Update creatChat Page
**File:** `frontend/src/views/creatChat/creatChat.vue`

Modify `createNewSession` to include `data_source_ids`:
```typescript
const selectedDataSources = settingsStore.settings.selectedDataSources || [];
sessionData.agent_config = {
  // ... existing config
  data_source_ids: selectedDataSources,
};
```

### Step 5: Update Chat Page
**File:** `frontend/src/views/chat/index.vue`

Ensure `data_source_ids` are passed in the stream request (already partially implemented in `sendMsg`).

## Testing Strategy
1. Verify "+" button appears in the input control bar
2. Test dropdown menu opens and closes correctly
3. Test DataSourceSelector modal opens when clicking "Select Data Source"
4. Verify data sources are fetched and displayed correctly
5. Test selection/deselection of data sources
6. Verify selected data sources appear as chips in the input field
7. Test that data_source_ids are passed to createSessions API
8. Test "Add New Data Source" navigation works

## Deliverables
- Updated settings store with data source state management
- New DataSourceSelector component
- Updated Input-field with "+" menu and data source selection
- Updated creatChat and chat pages to pass data_source_ids
