import { z } from "zod";

export const contextNameSchema = z.string().regex(/^[a-z][a-z0-9-]{0,31}$/);

const responseBase = {
  protocolVersion: z.literal(1),
  repository: z.string().min(1),
};

export const currentRepositorySchema = z.discriminatedUnion("mapped", [
  z
    .object({
      ...responseBase,
      mapped: z.literal(true),
      context: contextNameSchema,
    })
    .strict(),
  z
    .object({
      ...responseBase,
      mapped: z.literal(false),
      context: z.null(),
    })
    .strict(),
]);

export type CurrentRepository = z.infer<typeof currentRepositorySchema>;

export const contextListSchema = z
  .object({
    protocolVersion: z.literal(1),
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
