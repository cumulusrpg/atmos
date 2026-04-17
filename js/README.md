# atmos (JavaScript)

[![codecov (JS)](https://codecov.io/gh/cumulusrpg/atmos/branch/main/graph/badge.svg?flag=js)](https://codecov.io/gh/cumulusrpg/atmos)

JavaScript implementation of the atmos event-sourcing engine.

**Status:** nascent. The design lives at the repo root in [`../DESIGN.md`](../DESIGN.md); this package is growing toward it. Parity with the Go runtime is enforced informally for now, by keeping the two test suites telling the same story. Shared fixtures and a cross-runtime equivalence gate are a follow-up.

## Example

```js
import { createEngine } from '@cumulusrpg/atmos';

const engine = createEngine();

engine.registerState('todos', { items: [] });

engine.on('todo_added').updatesState('todos', (state, event) => ({
  items: [...state.items, { id: event.id, text: event.text, done: false }],
}));

engine.validate('todo_added', (event) => event.text.trim().length > 0);

engine.emit({ type: 'todo_added', id: '1', text: 'buy milk' });

engine.getState('todos');
// { items: [{ id: '1', text: 'buy milk', done: false }] }
```

See [`test/todo-app.test.js`](./test/todo-app.test.js) for the full story the engine tells today.

## Surface

- `registerState(name, initial)` — register a named state with its initial value
- `on(eventType).updatesState(stateName, reducer)` — attach a reducer
- `validate(eventType, fn)` — attach a validator; returning `false` rejects the event
- `emit(event)` — returns `true` if committed, `false` if rejected
- `getState(name)` — folds events through registered reducers
- `getEvents()` — the log

## Running tests

```sh
npm test              # run the suite
npm run test:coverage # with LCOV + text coverage report
```

Uses Node's built-in test runner (`node --test`, requires Node ≥ 20) and [c8](https://github.com/bcoe/c8) for coverage.
