# FabricError Definition

Standardizing on errors and problems are hard. We decided to use 
[**RFC7807**](https://tools.ietf.org/html/rfc7807) as our design for a problem structure. You don't have to use
this, however if you're looking for something that's ready-made and as standardized as can be, this may work for you.

---

## Type

A URI reference [RFC3986](https://datatracker.ietf.org/doc/html/rfc3986) that identifies the
problem type.  


## Title

A short, human-readable summary of the problem type.

## Status

The HTTP status code.

## Detail

A human-readable explanation specific to this occurrence of the problem

## Instance

A URI reference that identifies the specific occurrence of the problem.  It may or may not yield further 
information if de-referenced.
