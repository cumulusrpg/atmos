export function createEngine() {
  const events = [];
  const states = new Map(); // name -> { initial, reducers: Map<eventType, reducer> }
  const validators = new Map(); // eventType -> [validator fn]

  function registerState(name, initial) {
    states.set(name, { initial, reducers: new Map() });
  }

  function on(eventType) {
    return {
      updatesState(stateName, reducer) {
        const entry = states.get(stateName);
        entry.reducers.set(eventType, reducer);
      },
    };
  }

  function validate(eventType, fn) {
    if (!validators.has(eventType)) validators.set(eventType, []);
    validators.get(eventType).push(fn);
  }

  function emit(event) {
    const checks = validators.get(event.type) || [];
    for (const check of checks) {
      if (!check(event)) return false;
    }
    events.push(event);
    return true;
  }

  function getState(name) {
    const entry = states.get(name);
    let state = entry.initial;
    for (const event of events) {
      const reducer = entry.reducers.get(event.type);
      if (reducer) state = reducer(state, event);
    }
    return state;
  }

  function getEvents() {
    return events;
  }

  return { registerState, on, validate, emit, getState, getEvents };
}
