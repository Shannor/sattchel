# Learnings

> Will hold my learnings as I go along. Changes and pivots. Why I did the pivots and more.

## Pattern / Idea (2026-04-26)

---

The main Architecture Patterns attempted was Domain Driven Design (DDD) 
- Service Layer
    - Where the domain logic should live and stop. Things above this would be the UI/CLI/Consumer Layer.
- Data Mapper(s) Pattern
    - Focuses on how the data is returned from a Data Source. Data source could be a 3rd Party, Database, Local File system, etc.
    - Based on the pure description of this. It should be interacted with a Query Critera Constructor by the Domain Layer.
- Module Pattern
- Dependency Injection

I'm trying to use a Domain Model Pattern without a fully Object-Oriented Programming language. Therefore, some
patterns will mostly likely change based on how Golang works and its features and limitations.
Some patterns won't match at all to their textbook examples. But that's another fun challenge for me to understand
what patterns can work or can be modified to fit my needs.

## Pattern / Idea (2026-06-26)

---

Played around with DDD for about a month and a half and then worked pulled me away. 
I was starting to get a decent understanding of DDD and how it could be applied to my project. 
However, I realized that my project was not complex enough to warrant the use of DDD and doesn't really fit the use case.
I decided to pivot away from DDD and focus on other patterns and ideas.

My thought process now is that I'll need to simplify the logic, but some patterns actually are pretty clean after implementation.
I do like the idea of Data Mappers which are similar to what I used to call Data Access Objects (DAOs) back in the OOP days. 
They aren't exactly the same, but the idea is useful. I may keep them I haven't decided yet.

I've decided to care less about Styling and TUI stuff for now. The package is more annoying to work with than I expected. 
So ugly things for now until I'm getting useful functionality. 

Going forward, my goals are:
1. Focus on only two CLI groups Optimizely and what I'm calling "Tracker" for now. 
2. Find a different pattern that I've never done and work with it. I was looking at [ hexagonal architecture (ports and adapters) ](https://alistair.cockburn.us/hexagonal-architecture).
3. Still try to keep "Domain" models so that if I ever expand to a new source it still works. 
   1. Probably covered inside of hexagonal architecture probably. 

