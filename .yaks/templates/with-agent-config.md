# Problem
[What is wrong or missing, and where]

# Acceptance Criteria
- [ ] Specific, verifiable outcome
- [ ] Another verifiable outcome

# Files
- path/to/file.ts (what to change)

# Agent Config
Store this as a yak field before shaving:
    echo '{"model":"claude-sonnet-4-6","tools":["read","bash","edit","write"],"skills":[],"maxReviewRounds":1}' \
      | yx field "<yak name>" agent_config

Minimal read-only variant (docs, reviews):
    echo '{"tools":["read"],"maxReviewRounds":0}' | yx field "<yak name>" agent_config
