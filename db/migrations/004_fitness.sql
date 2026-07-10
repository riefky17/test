-- Fitness Tracker schema (home gym plans). Owned by fitness-svc.
--
-- Exercise/plan seed data below is intentionally generic-training-
-- principles content (progressive overload, PPL/upper-lower splits,
-- posterior-chain hypertrophy work), NOT a transcription of any paid
-- program's exact structure -- see README for the copyright note.
CREATE SCHEMA IF NOT EXISTS fitness;

CREATE TABLE IF NOT EXISTS fitness.exercises (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT NOT NULL UNIQUE,
    muscle_group TEXT NOT NULL,
    equipment    TEXT NOT NULL DEFAULT 'bodyweight'
);

CREATE TABLE IF NOT EXISTS fitness.plans (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    style       TEXT NOT NULL DEFAULT 'general' CHECK (style IN ('general', 'ppl-evidence-based', 'posterior-chain-focus')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fitness.plan_exercises (
    id          BIGSERIAL PRIMARY KEY,
    plan_id     BIGINT NOT NULL REFERENCES fitness.plans(id) ON DELETE CASCADE,
    exercise_id BIGINT NOT NULL REFERENCES fitness.exercises(id),
    day_label   TEXT NOT NULL,
    sets        INT NOT NULL DEFAULT 3,
    reps        TEXT NOT NULL DEFAULT '8-12',
    target_rpe  NUMERIC(3,1),
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS fitness.sessions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    plan_id     BIGINT REFERENCES fitness.plans(id) ON DELETE SET NULL,
    notes       TEXT,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fitness_plans_user ON fitness.plans(user_id);
CREATE INDEX IF NOT EXISTS idx_fitness_sessions_user ON fitness.sessions(user_id, performed_at DESC);

INSERT INTO fitness.exercises (name, muscle_group, equipment) VALUES
    ('Barbell Squat',        'legs',      'barbell'),
    ('Romanian Deadlift',    'posterior', 'barbell'),
    ('Hip Thrust',           'glutes',    'barbell'),
    ('Bench Press',          'push',      'barbell'),
    ('Overhead Press',       'push',      'barbell'),
    ('Barbell Row',          'pull',      'barbell'),
    ('Lat Pulldown',         'pull',      'cable'),
    ('Cable Kickback',       'glutes',    'cable'),
    ('Walking Lunge',        'legs',      'dumbbell'),
    ('Dumbbell Curl',        'pull',      'dumbbell'),
    ('Triceps Pushdown',     'push',      'cable'),
    ('Plank',                'core',      'bodyweight')
ON CONFLICT (name) DO NOTHING;
