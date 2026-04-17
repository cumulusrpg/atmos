// The story of atmos-js driving a simple todo app.
// Each test is a user action; each assertion is the state they should see.

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { createEngine } from '../src/engine.js';

function setupTodoApp() {
  const engine = createEngine();
  engine.registerState('todos', { items: [] });
  engine.on('todo_added').updatesState('todos', (state, event) => ({
    items: [...state.items, { id: event.id, text: event.text, done: false }],
  }));
  engine.on('todo_completed').updatesState('todos', (state, event) => ({
    items: state.items.map((t) =>
      t.id === event.id ? { ...t, done: true } : t
    ),
  }));
  engine.on('todo_removed').updatesState('todos', (state, event) => ({
    items: state.items.filter((t) => t.id !== event.id),
  }));
  engine.validate('todo_added', (event) => event.text.trim().length > 0);
  return engine;
}

test('a fresh todo app has no todos', () => {
  const app = setupTodoApp();
  assert.deepEqual(app.getState('todos'), { items: [] });
});

test('adding a todo makes it appear in the list', () => {
  const app = setupTodoApp();
  app.emit({ type: 'todo_added', id: '1', text: 'buy milk' });
  assert.deepEqual(app.getState('todos'), {
    items: [{ id: '1', text: 'buy milk', done: false }],
  });
});

test('completing a todo marks it done', () => {
  const app = setupTodoApp();
  app.emit({ type: 'todo_added', id: '1', text: 'buy milk' });
  app.emit({ type: 'todo_added', id: '2', text: 'walk dog' });
  app.emit({ type: 'todo_completed', id: '1' });
  assert.deepEqual(app.getState('todos'), {
    items: [
      { id: '1', text: 'buy milk', done: true },
      { id: '2', text: 'walk dog', done: false },
    ],
  });
});

test('removing a todo takes it off the list', () => {
  const app = setupTodoApp();
  app.emit({ type: 'todo_added', id: '1', text: 'buy milk' });
  app.emit({ type: 'todo_added', id: '2', text: 'walk dog' });
  app.emit({ type: 'todo_removed', id: '1' });
  assert.deepEqual(app.getState('todos'), {
    items: [{ id: '2', text: 'walk dog', done: false }],
  });
});

test('adding a todo with empty text is refused', () => {
  const app = setupTodoApp();
  app.emit({ type: 'todo_added', id: '1', text: '   ' });
  assert.deepEqual(app.getState('todos'), { items: [] });
});

test('replaying the same events produces the same state', () => {
  const events = [
    { type: 'todo_added', id: '1', text: 'buy milk' },
    { type: 'todo_added', id: '2', text: 'walk dog' },
    { type: 'todo_completed', id: '1' },
    { type: 'todo_removed', id: '2' },
  ];
  const appA = setupTodoApp();
  const appB = setupTodoApp();
  for (const e of events) appA.emit(e);
  for (const e of events) appB.emit(e);
  assert.deepEqual(appA.getState('todos'), appB.getState('todos'));
});

test('todos appear in the order they were added', () => {
  const app = setupTodoApp();
  app.emit({ type: 'todo_added', id: '1', text: 'buy milk' });
  app.emit({ type: 'todo_added', id: '2', text: 'walk dog' });
  app.emit({ type: 'todo_added', id: '3', text: 'write tests' });
  assert.deepEqual(app.getState('todos'), {
    items: [
      { id: '1', text: 'buy milk', done: false },
      { id: '2', text: 'walk dog', done: false },
      { id: '3', text: 'write tests', done: false },
    ],
  });
});
