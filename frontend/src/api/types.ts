import { z } from "zod";
import {
  FeedResponseSchema,
  ClusterResponseSchema,
  ClusterDetailSchema,
  ClusterSchema,
  ArticleSchema,
  SourceSchema,
  HomepageClusterSectionSchema,
  HomepageSourceGroupSchema,
  HomepageSourceSchema,
  HomepageArticleSchema,
  LatestArticleSchema,
  LatestArticlesResponseSchema,
} from "./schemas";

export type FeedResponse = z.infer<typeof FeedResponseSchema>;
export type ClusterResponse = z.infer<typeof ClusterResponseSchema>;
export type ClusterDetail = z.infer<typeof ClusterDetailSchema>;
export type Cluster = z.infer<typeof ClusterSchema>;
export type Article = z.infer<typeof ArticleSchema>;
export type Source = z.infer<typeof SourceSchema>;
export type HomepageClusterSection = z.infer<typeof HomepageClusterSectionSchema>;
export type HomepageSourceGroup = z.infer<typeof HomepageSourceGroupSchema>;
export type HomepageSource = z.infer<typeof HomepageSourceSchema>;
export type HomepageArticle = z.infer<typeof HomepageArticleSchema>;
export type LatestArticle = z.infer<typeof LatestArticleSchema>;
export type LatestArticlesResponse = z.infer<typeof LatestArticlesResponseSchema>;
