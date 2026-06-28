# Tracker

> Better name TBD

The point of this tool is an attempt to see how I can merge two problems I've been having.
1. How goals that the team is working on are linked together. 
2. Who is working on which goals and what is the "impact" of those goals.


Basically, I want the view of a Mind Map where I can see how the goals all relate and 
unlock some new idea or insight we can work towards. 
At the same time I also want to add a simple X/Y ratio of Effort/Impact for each goal. 
Because, I want to see that even five hard + low impact goals can have a big impact if they "unlock" 
a high impact goal, even if that's not immediately obvious.

Since I'm learning CLI stuff I figured I'd be able to make a simple CLI tool to help me with this so it can be quick
to add to it but easily transferable to something else data wise. 

Domain Models / Concepts I think I need right now:
- Goal
- Link 
- Member 
- Effort
- Impact 


So this would be a tree structure (time to pull out my college degree I guess).

A Goal will have a relationship to other goals. With one Goal being the root with no parent links.
I'm thinking the rule will be no cycles and a goal can only have one parent.
A Memeber can be attached to multiple goals.
A Goal will have effort and impact attached to it. 

Relationships between goals can be:
- Required
- Optional
- Preferred
