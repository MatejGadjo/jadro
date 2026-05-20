import { z } from "zod";

export const SourceSchema = z.object({
  id: z.number(),
  name: z.string(),
  base_url: z.string(),
  rss_url: z.string(),
  logo_url: z.string().optional(),
});

export const ClusterSchema = z.object({
  id: z.number(),
  headline: z.string(),
  display_title: z.string().optional(),
  summary: z.string(),
  topic: z.string(),
  score: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const ArticleSchema = z.object({
  id: z.number(),
  url: z.string(),
  image_url: z.string().optional(),
  title: z.string(),
  snippet: z.string(),
  published_at: z.string().nullable().optional(),
  fetched_at: z.string(),
  source: SourceSchema,
});

export const HomepageSourceSchema = z.object({
  id: z.number(),
  name: z.string(),
  base_url: z.string(),
  rss_url: z.string(),
  logo_url: z.string().optional(),
  domain: z.string(),
});

export const HomepageArticleSchema = z.object({
  id: z.number(),
  url: z.string(),
  image_url: z.string().optional(),
  title: z.string(),
  snippet: z.string(),
  published_at: z.string().nullable().optional(),
  fetched_at: z.string(),
});

export const HomepageSourceGroupSchema = z.object({
  source: HomepageSourceSchema,
  articles: z.array(HomepageArticleSchema),
});

export const HomepageClusterSectionSchema = z.object({
  category: z.string().optional(),
  cluster: ClusterSchema,
  sources: z.array(HomepageSourceGroupSchema),
});

export const FeedResponseSchema = z.object({
  feed_version: z.number().optional().default(0),
  clusters: z.array(ClusterSchema),
  featured_clusters: z.array(HomepageClusterSectionSchema).optional().default([]),
  top_clusters_by_category: z.array(HomepageClusterSectionSchema).optional().default([]),
});

export const ClusterResponseSchema = z.object({
  cluster: ClusterSchema,
  articles: z.array(ArticleSchema),
});

export const LatestArticleSchema = z.object({
  id: z.number(),
  url: z.string(),
  title: z.string(),
  topic: z.string(),
  published_at: z.string().nullable().optional(),
  fetched_at: z.string(),
  source_name: z.string(),
  source_logo_url: z.string().nullable().optional(),
});

export const LatestArticlesResponseSchema = z.object({
  articles: z.array(LatestArticleSchema),
});

