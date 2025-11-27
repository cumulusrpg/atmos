Feature: Snapshot Support for E2E Testing
  As a developer writing E2E tests
  I want to seed specific game states as snapshots
  So that I can test UI interactions without replaying many events

  Background:
    Given an engine with snapshot-capable repository
    And a "game" state with default values:
      | field | value |
      | score | 0     |
      | level | 1     |

  # ==========================================================================
  # Snapshot Storage
  # ==========================================================================

  Scenario: Store and retrieve a snapshot
    When I set a snapshot for "game" state with:
      | field | value |
      | score | 100   |
      | level | 5     |
    Then the snapshot for "game" should exist
    And the snapshot for "game" should contain:
      | field | value |
      | score | 100   |
      | level | 5     |

  Scenario: Retrieve non-existent snapshot
    Then the snapshot for "game" should not exist

  Scenario: Clear an existing snapshot
    Given I set a snapshot for "game" state with:
      | field | value |
      | score | 100   |
    When I clear the snapshot for "game"
    Then the snapshot for "game" should not exist

  # ==========================================================================
  # Snapshot-Aware State Projection
  # ==========================================================================

  Scenario: Project state without snapshot uses initial state
    When I emit a "score" event with points 10
    And I emit a "score" event with points 20
    Then the "game" state should have:
      | field | value |
      | score | 30    |
      | level | 1     |

  Scenario: Project state with snapshot starts from snapshot
    Given I set a snapshot for "game" state with:
      | field | value |
      | score | 100   |
      | level | 5     |
    When I emit a "score" event with points 10
    And I emit a "score" event with points 20
    Then the "game" state should have:
      | field | value |
      | score | 130   |
      | level | 5     |

  Scenario: Partial snapshot merges over defaults
    Given I set a snapshot for "game" state with:
      | field | value |
      | score | 50    |
    Then the "game" state should have:
      | field | value |
      | score | 50    |
      | level | 1     |

  # ==========================================================================
  # Non-Snapshot Repository Behavior
  # ==========================================================================

  Scenario: Engine without snapshot repository works normally
    Given an engine with standard repository
    And a "game" state with default values:
      | field | value |
      | score | 0     |
      | level | 1     |
    When I emit a "score" event with points 10
    Then the "game" state should have:
      | field | value |
      | score | 10    |
      | level | 1     |

  Scenario: Setting snapshot on non-snapshot repository returns error
    Given an engine with standard repository
    And a "game" state with default values:
      | field | value |
      | score | 0     |
    When I try to set a snapshot for "game" state
    Then I should receive a snapshot error

  # ==========================================================================
  # E2E Testing Scenario
  # ==========================================================================

  Scenario: Seed game state for E2E testing
    Given I set a snapshot for "game" state with:
      | field | value |
      | score | 500   |
      | level | 3     |
    Then the "game" state should have:
      | field | value |
      | score | 500   |
      | level | 3     |
    When I emit a "score" event with points 50
    Then the "game" state should have:
      | field | value |
      | score | 550   |
      | level | 3     |

  Scenario: Check if snapshot exists
    Then the engine should report no snapshot for "game"
    When I set a snapshot for "game" state with:
      | field | value |
      | score | 100   |
    Then the engine should report a snapshot exists for "game"
    When I clear the snapshot for "game"
    Then the engine should report no snapshot for "game"
