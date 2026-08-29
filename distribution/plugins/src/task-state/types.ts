export type StepStatus = "pending" | "in_progress" | "completed" | "blocked";

export interface NewStep {
  id: string;
  title: string;
  verification?: string;
}

export interface TaskPlanStep {
  id: string;
  title: string;
  verification?: string;
  status: StepStatus;
  evidence?: string;
}

export interface TaskPlan {
  id: string;
  sessionId: string;
  goal: string;
  steps: TaskPlanStep[];
  createdAt: string;
  updatedAt: string;
  version: 1;
}

export const TASK_PLAN_LIMITS = {
  minSteps: 3,
  maxSteps: 7,
  goalChars: 1000,
  stepIDChars: 100,
  stepTitleChars: 300,
  verificationChars: 500,
  evidenceChars: 1000,
} as const;
