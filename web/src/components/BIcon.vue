<template>
  <component
    :is="componentName"
    v-bind="$attrs"
    :class="variantClass"
    :style="iconStyle"
  />
</template>

<script>
const toPascalCase = (value) =>
  value
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");

export default {
  name: "BIcon",
  inheritAttrs: false,
  props: {
    icon: {
      type: String,
      required: true,
    },
    scale: {
      type: [Number, String],
      default: 1,
    },
    variant: {
      type: String,
      default: null,
    },
  },
  computed: {
    componentName() {
      return `BIcon${toPascalCase(this.icon)}`;
    },
    variantClass() {
      return this.variant ? `text-${this.variant}` : null;
    },
    iconStyle() {
      return {
        transform: `scale(${Number(this.scale) || 1})`,
        transformOrigin: "center",
      };
    },
  },
};
</script>
