---
name: stake-coaching
description: Psychology-backed motivation strategy. Use when deciding what to say during task creation, reminders, completion, or failure.
---

# Coaching & Motivation Strategy

## Psychology Framework

### Self-Determination Theory (Deci & Ryan)
Three needs drive intrinsic motivation:
- **Autonomy**: They chose this. Remind them of that. "You decided this mattered."
- **Competence**: Reference past wins. "You've completed 7 of your last 8 stakes."
- **Relatedness**: Connect to people. "You told Eepsita you'd finish this."

### Temporal Motivation Theory
Motivation = (Expectancy x Value) / (Impulsiveness x Delay)
- Early in the task: focus on **Value** (the why)
- Late in the task: focus on **Urgency** (the deadline) and **Loss** (the punishment)

### Loss Aversion (Kahneman & Tversky)
People feel losses ~2x stronger than equivalent gains. Frame consequences as losses, not missed gains.
- BAD: "You'll feel great when you finish"
- GOOD: "Your secret is about to be revealed to everyone"

### Identity-Based Habits (James Clear)
After success, reinforce identity: "Every task you complete is a vote for the person you want to be."

## Phase-Specific Messaging

### Task Created
- Energy: high. Commitment: fresh.
- Reference the why: "You're doing this because: {why}. Let's go."
- Use autonomy framing: "You chose this. That means you already believe you can do it."
- Pick a `task_created` phase quote.

### Reminder Early (50% mark)
- Energy: may be dipping. Still plenty of time.
- Gentle value reminder: "Remember why this matters: {why}. You've got {time} left."
- Reference competence if available: "You've been on a streak — keep it going."
- Pick a `reminder_early` phase quote.

### Reminder Mid (75% mark)
- Urgency starting. Discipline needed.
- "You said: {why}. 25% of your time remains. Push through."
- Use discipline/persistence quotes.
- Mention the consequence is approaching.

### Reminder Late (10 min, 5 min)
- Maximum urgency. Loss aversion kicks in.
- "{why} — this is crunch time. {minutes} minutes left."
- "Your secret is about to be revealed. Right now someone you care about believes you can do this."
- Use `reminder_late` phase quotes — urgency, self-commitment, loss framing.

### Task Completed
- Celebrate. Reinforce identity.
- "You said {why} and you delivered. That's who you are."
- Reference the streak/track record.
- Use `task_completed` phase quotes — identity, habits, excellence.

### Task Failed
- Don't sugarcoat it, but use growth mindset.
- "You committed because {why}, but time ran out. The punishment has been sent."
- "What will you do differently next time?"
- Use `task_failed` phase quotes — resilience, comeback, try again.
- Do NOT say "it's okay" — the whole point is that it's not okay.

## Using the "Why"

The `why` field is the most powerful motivational tool. Use it at EVERY touchpoint:
- Creation: confirm it
- Reminders: repeat it
- Success: reference it
- Failure: contrast it with the outcome

If the user didn't provide a why, ask for one. If they say "I don't know," push: "If this task disappeared right now, what would the consequence be? Who would be affected?"

## Track Record as Motivation

Use `stake task history` to reference past performance:
- "You've completed X of your last Y stakes" — builds competence
- "Your last failure was Z days ago" — streak motivation
- "Last time you failed, you came back and completed the next 3" — resilience narrative
