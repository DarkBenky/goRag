# goRag

goRag is a hobby project to create an optimized RAG server in Golang
using the Go 1.27 SIMD package with configurable RAM usage.

## Goal
Make fast and memory efficient RAG that can be deployed on different hardware

## Pipeline

0. Select the number of Groups (**Groups are always in RAM to speed up inference**) and initialize each Group with a
   random centroid vector.

1. For each embedding:

   Compute dot product between the embedding and each Group centroid:

       MulAdd(Embedding, GroupCentroid) => reduce(sum)

2. Assign each embedding to the best-matching Group.

3. Recompute each Group's centroid from its assigned embeddings:

       GroupCentroid =
           sum(assigned embeddings element-wise) / number of embeddings

4. Repeat steps 1–3 until the groups/centroids converge.

5. Store each Group's embeddings as a separate chunk on disk.
   Keep the Group size so memory requirements can be determined.

6. Search for searchEmbedding:

   6.1 Compute dot product between searchEmbedding and every Group centroid:

       MulAdd(searchEmbedding, GroupCentroid) => reduce(sum)

   6.2 Select the top K best-matching Groups.

   6.3 Ensure those chunks are in memory:
       - if already loaded, use them
       - otherwise unload chunks if necessary and load the required chunks

7. Compute dot product between searchEmbedding and every embedding
   inside the selected chunks:

       MulAdd(searchEmbedding, sampleEmbedding) => reduce(sum)

   Keep the N samples with the highest score.

8. Return the top N samples.