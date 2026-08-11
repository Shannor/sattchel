export class Layout {
	static nodeWidth = 240;
	static nodeHeight = 110;
	static cpuWidth = 300;
	static cpuHeight = 120;
	static spacingX = 120;
	static spacingY = 32;

	static compute(_goalsMap, roots, rootChildSides = {}) {
		if (roots.length === 0) return;

		// Position the main root (CPU) in the exact center of the map
		const root = roots[0];
		root.side = "root";
		root.x = 2000 - Layout.cpuWidth / 2;
		root.y = 1500 - Layout.cpuHeight / 2;

		// Split children of the root into left and right sides.
		// Sides are assigned once per goal ID and then frozen in rootChildSides
		// so that re-renders after a drag-drop don't reshuffle the tree.
		const leftChildren = [];
		const rightChildren = [];

		function setSide(node, side) {
			node.side = side;
			node.children.forEach((child) => setSide(child, side));
		}

		// Count how many are already pinned to each side so new arrivals
		// continue to balance onto the shorter side.
		let pinnedLeft = Object.values(rootChildSides).filter(
			(s) => s === "left",
		).length;
		let pinnedRight = Object.values(rootChildSides).filter(
			(s) => s === "right",
		).length;

		root.children.forEach((child) => {
			if (!(child.id in rootChildSides)) {
				// New child: assign to whichever side is currently shorter.
				// Ties go left first, matching the original even/odd behaviour.
				const side = pinnedLeft <= pinnedRight ? "left" : "right";
				rootChildSides[child.id] = side;
				if (side === "left") pinnedLeft++;
				else pinnedRight++;
			}
			const side = rootChildSides[child.id];
			setSide(child, side);
			if (side === "left") {
				leftChildren.push(child);
			} else {
				rightChildren.push(child);
			}
		});

		// 1. Calculate subtree heights post-order
		function calculateHeight(node) {
			if (node.children.length === 0) {
				node.subtreeHeight = Layout.nodeHeight;
				return Layout.nodeHeight;
			}
			let totalHeight = 0;
			node.children.forEach((child) => {
				totalHeight += calculateHeight(child);
			});
			totalHeight += (node.children.length - 1) * Layout.spacingY;
			node.subtreeHeight = Math.max(Layout.nodeHeight, totalHeight);
			return node.subtreeHeight;
		}

		root.children.forEach((child) => calculateHeight(child));

		// 2. Position nodes recursively
		function positionNode(node, x, yTop) {
			node.x = x;

			if (node.children.length === 0) {
				node.y = yTop + (node.subtreeHeight - Layout.nodeHeight) / 2;
				return;
			}

			let currentY = yTop;
			node.children.forEach((child) => {
				const nextX =
					node.side === "left"
						? x - Layout.nodeWidth - Layout.spacingX
						: x + Layout.nodeWidth + Layout.spacingX;
				positionNode(child, nextX, currentY);
				currentY += child.subtreeHeight + Layout.spacingY;
			});

			const firstChild = node.children[0];
			const lastChild = node.children[node.children.length - 1];
			node.y = (firstChild.y + lastChild.y) / 2;
		}

		// Calculate total height of each side to center them vertically around Y=1500
		const leftSideHeight =
			leftChildren.reduce((acc, child) => acc + child.subtreeHeight, 0) +
			(leftChildren.length - 1) * Layout.spacingY;
		const rightSideHeight =
			rightChildren.reduce((acc, child) => acc + child.subtreeHeight, 0) +
			(rightChildren.length - 1) * Layout.spacingY;

		// Position Left Subtrees
		if (leftChildren.length > 0) {
			const leftYTop = 1500 - leftSideHeight / 2;
			let currentY = leftYTop;
			leftChildren.forEach((child) => {
				const firstX =
					2000 - Layout.cpuWidth / 2 - Layout.spacingX - Layout.nodeWidth;
				positionNode(child, firstX, currentY);
				currentY += child.subtreeHeight + Layout.spacingY;
			});
		}

		// Position Right Subtrees
		if (rightChildren.length > 0) {
			const rightYTop = 1500 - rightSideHeight / 2;
			let currentY = rightYTop;
			rightChildren.forEach((child) => {
				const firstX = 2000 + Layout.cpuWidth / 2 + Layout.spacingX;
				positionNode(child, firstX, currentY);
				currentY += child.subtreeHeight + Layout.spacingY;
			});
		}

		// Extra roots layout (if any) placed far below
		let extraY =
			1500 +
			Math.max(leftSideHeight, rightSideHeight, Layout.cpuHeight) / 2 +
			300;
		for (let i = 1; i < roots.length; i++) {
			const extraRoot = roots[i];
			extraRoot.side = "right";
			calculateHeight(extraRoot);
			positionNode(
				extraRoot,
				2000 + Layout.cpuWidth / 2 + Layout.spacingX,
				extraY,
			);
			extraY += extraRoot.subtreeHeight + 300;
		}
	}
}
