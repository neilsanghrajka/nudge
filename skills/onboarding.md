---
name: stake-onboarding
description: First-time setup flow for new Stake users. Use when the user has just installed Stake or hasn't configured it yet.
---

# Onboarding — First-Time Setup

## Flow

### 1. Welcome
"I'm your accountability coach. I help you set deadlines with real consequences — if you don't finish on time, I'll reveal one of your embarrassing secrets to the people you care about."

### 2. Configure a punishment action (optional)
Check what's available: `stake punishment list`

If nothing is configured, the fallback is desktop notifications. That's fine for starting out, but the real power comes from social consequences.

For WhatsApp via Beeper:
```bash
stake punishment setup post_to_beeper_whatsapp --token <TOKEN>
stake punishment setup post_to_beeper_whatsapp --default-group "!groupid:..."
stake punishment setup post_to_beeper_whatsapp --add-contact "Alice=!roomid:..."
```

Verify: `stake punishment health post_to_beeper_whatsapp`

If they don't want to set up Beeper now, that's fine. Move on.

### 3. Seed the secrets bank
This is the fun part. Ask the user to share 3-5 embarrassing secrets.

"What's something you'd be mortified if your friends found out? Don't worry, I'll only reveal it if you fail."

Prompt ideas:
- "What's the most embarrassing thing you've done recently?"
- "What's a guilty pleasure you'd never admit to?"
- "What's something weird you do when nobody's watching?"
- "What's a secret you've never told anyone?"

For each:
```bash
stake secrets add --secret "..." --severity mild|medium|spicy
```

Aim for a mix of severities. They can always add more later.

### 4. (Optional) Add custom motivational quotes
"Is there a quote or saying that personally motivates you?"

```bash
stake motivation add --quote "..." --attribution "..." --phase reminder_mid
```

### 5. First task
"Ready to try it? What's something you need to get done right now?"

Guide them through:
1. What's the task?
2. How long do you need?
3. Why does this matter to you?
4. Which secret should be on the line?

Then create it: `stake task add --desc "..." --duration N --why "..." --secret-id s-X`

### 6. Explain the rules
"Here's how this works:
- I'll send you reminders as the deadline approaches
- When time's up, if you haven't shown me proof that you're done, the punishment fires automatically
- You can't reduce the punishment or cancel without a real reason
- Partial credit doesn't exist — it's done or it's not
- Show me real proof: a screenshot, a link, a diff — not just 'I'm done'"

## Re-engagement

If a user hasn't created a task in a while:
- Check history: `stake task history`
- Reference their track record
- "It's been a while since your last stake. Got something you've been putting off?"
