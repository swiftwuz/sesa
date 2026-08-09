import { z } from "zod";

export const contextNameSchema = z.string().regex(/^[a-z][a-z0-9-]{0,31}$/);

const contextSetSchema = z
  .array(contextNameSchema)
  .refine((contexts) => new Set(contexts).size === contexts.length, {
    message: "Context names must be unique",
  });

const responseBase = {
  protocolVersion: z.literal(2),
  repository: z.string().min(1),
};

export const currentRepositorySchema = z.discriminatedUnion("mapped", [
  z
    .object({
      ...responseBase,
      mapped: z.literal(true),
      contexts: contextSetSchema.refine((contexts) => contexts.length > 0),
    })
    .strict(),
  z
    .object({
      ...responseBase,
      mapped: z.literal(false),
      contexts: contextSetSchema.refine((contexts) => contexts.length === 0),
    })
    .strict(),
]);

export type CurrentRepository = z.infer<typeof currentRepositorySchema>;

export const contextListSchema = z
  .object({
    protocolVersion: z.literal(2),
    contexts: z.array(contextNameSchema),
  })
  .strict();

export type ContextList = z.infer<typeof contextListSchema>;

export function parseCurrentRepository(output: string): CurrentRepository {
  return currentRepositorySchema.parse(JSON.parse(output));
}

export function parseContextList(output: string): ContextList {
  return contextListSchema.parse(JSON.parse(output));
}

export function parseActiveContext(
  value: string | undefined,
): string | undefined {
  const result = contextNameSchema.safeParse(value);
  return result.success ? result.data : undefined;
}
