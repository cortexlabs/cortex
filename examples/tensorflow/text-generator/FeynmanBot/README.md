# Write Your Own Richard Feynman Lecture With GPT-2

In this example, we're going to train a GPT-2 model using Richard Feynman's famous lectures, and then deploy the model via Cortex.

The bulk of this tutorial takes place in [this notebook](https://colab.research.google.com/drive/1zCN68S0-WtSFI-Lsp6ebeNk3fhcwsMWj), which you can also find a copy of in this repo as `feynman.ipynb`. This notebook will take you through training the model, exporting it for serving, uploading it to AWS, and deploying it with Cortex.

Once you finish the notebook, you will be able to generate imitation-Feynman lectures, like this:
# Example Feynman GPT-2 Lecture

_According to quantum mechanics, the rate at which new electrical and chemical phenomena are produced is proportional to the fraction of the total periodicity of the phenomena. That is to say, the fraction of the periodic time which is true is much smaller than the fraction of the time which is false. The physical physicist Gediminas proposed a new law, the Mechanics of Light, which explains why the rate of change of the reflected light of a molecule is so small. The new law is independent of the previous one, and can be applied to the history of light as we know it. This short essay analyzes the most important ideas in the theory of light and argues that they're not isolated matters—that all light can be represented in symbols on the light sabre. It is a statement of the law of the phase of light which gives the name of this law to show that it is possible to write it in several different ways._

_Why does the law of phase of light give a blue ray? Because the atoms in the water which we drink become “blue” as they age. As the age passes, more and more of the oxygen and hydrogen are in the water and the water becomes “hot.” In some words, as the age goes on more and more of the atoms are lost in the water and the water turns into liquid water. If we could “see” the blue rays of the atoms in the water, we might guess at the age of the plant, but for most atomic problems, we cannot figure out the law of phase change. This is an abstract problem, and we do not have the time to finish it. There is an analog of this to the physicist in discussing weeds. He says: “You do not know the history of the plants until you have seen some of their flowers. When they are finished you know how large they are!” Plants are no longer secrets; instead, these “hidden” plants are common. Plants have names. They are sometimes called cacti, genera, flowers, stems, veins, or serrations, because of their mutual relations among roots and meningles. Plants are sometimes called fasciids, for fondness for blood, or even cactus. Plants are sometimes called bracts, or corving. There are many different kinds of stems. A branch or a branch-stem is analogous to a branch or a point. But a simpleton branch or a fixed point is not equivalent. If we extend the branch of a plant to the manifold varied possibilities, we get branches that are compound inclined, like a cork, with a smooth, flat, straight “line” that divides into two “pillars” or “flutters.” If we want to type something beautiful on a plant, we must make it plane, rather than straight._


# Prerequisites
In order to build your own Feynman (or whatever writer your choose) API, you'll need to do a few things first:

1. Install [Cortex](https://www.cortex.dev/install)
2. Download the text you want to train your GPT-2 model with.
3. Create an S3 bucket where you'd like your model to be hosted.

Note that you do not need to format your text in any special way. Simply paste all the text you'd like GPT-2 to train on into a plain .txt file.

# Notes on the Notebook

The [notebook](https://colab.research.google.com/drive/1zCN68S0-WtSFI-Lsp6ebeNk3fhcwsMWj) you'll be running should be self-explanatory, but in case you get stuck, here are a few things to keep in mind.

First, different points in the notebook will require manual inputs from you. You will have to enter your AWS details, and you will have to enter a success code in order to mount your Google Drive to Colab.

Secondly, messing with any file or directory names can have downstream impacts, so be prepared to squash some minor bugs if you do so.

Lastly, Cortex is an open source project whose files and folders change frequently. If any link is broken, or any file inaccessible, just leave open an issue here and we will fix it ASAP.

That's everything! If you generate anything particularly cool, feel free to share it with us on Twitter [@cortex_deploy](https://twitter.com/cortex_deploy).
