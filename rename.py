import re

with open('internal/app/evaluate_funding_candidate_deep.go', 'r') as f:
    code = f.read()

# Replace ecd with efcd
code = re.sub(r'\becd', 'efcd', code)
code = re.sub(r'phase103', 'phase107', code)
code = re.sub(r'Phase103', 'Phase107', code)

with open('internal/app/evaluate_funding_candidate_deep.go', 'w') as f:
    f.write(code)
